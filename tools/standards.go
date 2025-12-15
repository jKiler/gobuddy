package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultCacheDuration = 336 * time.Hour // 2 weeks
)

// StandardsSource represents a source of coding standards.
type StandardsSource struct {
	Type     string   `json:"type" jsonschema:"type of source (local, git, or url)"`
	Location string   `json:"location" jsonschema:"file path for local, git URL for git, or HTTP(S) URL for url sources"`
	Files    []string `json:"files,omitempty" jsonschema:"specific markdown files to fetch from the source (not used for url type)"`
	Branch   string   `json:"branch,omitempty" jsonschema:"git branch or tag to checkout (defaults to main)"`
	Priority int      `json:"priority" jsonschema:"priority of this source (lower number = higher priority)"`
}

// StandardsInput is the input for the fetch_coding_standards tool.
type StandardsInput struct {
	Preset        string            `json:"preset,omitempty" jsonschema:"preset name (e.g., google-go, uber-go, effective-go). If specified, uses preset sources instead of custom sources"`
	Sources       []StandardsSource `json:"sources,omitempty" jsonschema:"list of sources to fetch standards from (uses default sources if empty and no preset specified)"`
	IncludeSource bool              `json:"include_source,omitempty" jsonschema:"include source information in output"`
	CacheDir      string            `json:"cache_dir,omitempty" jsonschema:"directory to cache cloned repos (defaults to system temp dir)"`
}

// StandardsOutput is the output for the fetch_coding_standards tool.
type StandardsOutput struct {
	Standards string `json:"standards" jsonschema:"the aggregated coding standards content"`
}

// Standards fetches coding standards from multiple sources and aggregates them.
func Standards(ctx context.Context, req *mcp.CallToolRequest, input StandardsInput) (
	*mcp.CallToolResult,
	StandardsOutput,
	error,
) {
	// Set defaults
	if input.CacheDir == "" {
		input.CacheDir = filepath.Join(os.TempDir(), "gobuddy-standards-cache")
	}

	// Determine sources
	if input.Preset != "" {
		// Use preset sources
		presets := getPresets()
		sources, ok := presets[input.Preset]
		if !ok {
			available := make([]string, 0, len(presets))
			for name := range presets {
				available = append(available, name)
			}
			sort.Strings(available)
			return errorResult(fmt.Errorf("unknown preset: %s. Available presets: %v", input.Preset, available))
		}
		input.Sources = sources
	} else if len(input.Sources) == 0 {
		// Use default sources if no preset and no custom sources
		input.Sources = getDefaultSources()
	}

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(input.CacheDir, 0755); err != nil {
		return errorResult(fmt.Errorf("failed to create cache directory: %w", err))
	}

	// Sort sources by priority (lower number = higher priority)
	sort.Slice(input.Sources, func(i, j int) bool {
		return input.Sources[i].Priority < input.Sources[j].Priority
	})

	var aggregated strings.Builder
	successCount := 0

	for _, source := range input.Sources {
		content, err := fetchSource(ctx, source, input.CacheDir)
		if err != nil {
			// Log error but continue with other sources
			aggregated.WriteString(fmt.Sprintf("\n<!-- Failed to fetch from %s: %v -->\n", source.Location, err))
			continue
		}

		if content != "" {
			successCount++
			if input.IncludeSource {
				aggregated.WriteString(fmt.Sprintf("\n---\n## Source: %s\n**Type:** %s | **Priority:** %d\n\n",
					source.Location, source.Type, source.Priority))
			}
			aggregated.WriteString(content)
			aggregated.WriteString("\n\n")
		}
	}

	if successCount == 0 {
		return errorResult(fmt.Errorf("failed to fetch standards from any source"))
	}

	result := aggregated.String()
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, StandardsOutput{Standards: result}, nil
}

// getDefaultSources returns the default sources for coding standards.
// Note: The git source is an example and will fail unless customized to point
// to an actual repository. Users should specify their own sources or use presets.
func getDefaultSources() []StandardsSource {
	return []StandardsSource{
		{
			Type:     "local",
			Location: ".",
			Files:    []string{"CODING_STANDARDS.md", ".gostandards/STANDARDS.md", "STANDARDS.md"},
			Priority: 1,
		},
		{
			Type:     "git",
			Location: "git@github.com:your-org/go-standards.git", // Example placeholder
			Files:    []string{"STANDARDS.md", "GO_GUIDELINES.md"},
			Branch:   "main",
			Priority: 10,
		},
	}
}

// getPresets returns a map of preset names to their StandardsSource configurations.
func getPresets() map[string][]StandardsSource {
	return map[string][]StandardsSource{
		"google-go": {
			{
				Type:     "url",
				Location: "https://raw.githubusercontent.com/google/styleguide/gh-pages/go/guide.md",
				Priority: 1,
			},
			{
				Type:     "url",
				Location: "https://raw.githubusercontent.com/google/styleguide/gh-pages/go/decisions.md",
				Priority: 2,
			},
			{
				Type:     "url",
				Location: "https://raw.githubusercontent.com/google/styleguide/gh-pages/go/best-practices.md",
				Priority: 3,
			},
		},
		"uber-go": {
			{
				Type:     "url",
				Location: "https://raw.githubusercontent.com/uber-go/guide/master/style.md",
				Priority: 1,
			},
		},
		"effective-go": {
			{
				Type:     "url",
				Location: "https://go.dev/doc/effective_go",
				Priority: 1,
			},
		},
	}
}

// fetchSource fetches content from a single source.
func fetchSource(ctx context.Context, source StandardsSource, cacheDir string) (string, error) {
	switch source.Type {
	case "local":
		return fetchLocalSource(source)
	case "git":
		return fetchGitSource(ctx, source, cacheDir)
	case "url":
		return fetchURLSource(ctx, source, cacheDir)
	default:
		return "", fmt.Errorf("unknown source type: %s", source.Type)
	}
}

// fetchLocalSource fetches content from local files.
func fetchLocalSource(source StandardsSource) (string, error) {
	var content strings.Builder

	for _, file := range source.Files {
		fullPath := filepath.Join(source.Location, file)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			// File doesn't exist, try next one
			continue
		}

		content.WriteString(fmt.Sprintf("### %s\n\n", file))
		content.Write(data)
		content.WriteString("\n\n")
	}

	if content.Len() == 0 {
		return "", fmt.Errorf("no local files found")
	}

	return content.String(), nil
}

// fetchGitSource fetches content from a git repository.
func fetchGitSource(ctx context.Context, source StandardsSource, cacheDir string) (string, error) {
	branch := source.Branch
	if branch == "" {
		branch = "main"
	}

	// Create a unique directory name based on the repo URL
	hash := sha256.Sum256([]byte(source.Location))
	repoDir := filepath.Join(cacheDir, hex.EncodeToString(hash[:])[:16])

	// Check if cache exists and is fresh
	if needsRefresh(repoDir) {
		// Remove old cache if it exists; ignore errors as we'll clone fresh anyway
		os.RemoveAll(repoDir)

		// Clone the repository
		cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", "--branch", branch, source.Location, repoDir)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git clone failed: %w\nOutput: %s", err, string(output))
		}
	} else {
		// Cache is fresh, just pull latest changes
		cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "pull", "--ff-only")
		// Ignore pull errors - use existing cache if pull fails
		_ = cmd.Run()
	}

	// Update cache timestamp
	touchCacheTimestamp(repoDir)

	// Read the specified files
	var content strings.Builder
	for _, file := range source.Files {
		fullPath := filepath.Join(repoDir, file)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			// File doesn't exist in repo, try next one
			continue
		}

		content.WriteString(fmt.Sprintf("### %s\n\n", file))
		content.Write(data)
		content.WriteString("\n\n")
	}

	if content.Len() == 0 {
		return "", fmt.Errorf("no files found in git repository")
	}

	return content.String(), nil
}

// fetchURLSource fetches content from an HTTP(S) URL.
func fetchURLSource(ctx context.Context, source StandardsSource, cacheDir string) (string, error) {
	// Validate URL
	if !strings.HasPrefix(source.Location, "http://") && !strings.HasPrefix(source.Location, "https://") {
		return "", fmt.Errorf("url source must start with http:// or https://")
	}

	// Create cache key from URL
	hash := sha256.Sum256([]byte(source.Location))
	cacheFile := filepath.Join(cacheDir, hex.EncodeToString(hash[:])[:16]+".md")

	// Check cache freshness
	if !needsRefresh(filepath.Dir(cacheFile)) {
		// Try to read from cache
		if data, err := os.ReadFile(cacheFile); err == nil {
			return string(data), nil
		}
	}

	// Fetch from URL
	req, err := http.NewRequestWithContext(ctx, "GET", source.Location, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Cache the content
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err == nil {
		os.WriteFile(cacheFile, data, 0644)
		touchCacheTimestamp(filepath.Dir(cacheFile))
	}

	return string(data), nil
}

// needsRefresh checks if the cache needs to be refreshed.
func needsRefresh(repoDir string) bool {
	timestampFile := filepath.Join(repoDir, ".gobuddy-cache-timestamp")
	info, err := os.Stat(timestampFile)
	if err != nil {
		return true // Cache doesn't exist or can't be read
	}

	return time.Since(info.ModTime()) > defaultCacheDuration
}

// touchCacheTimestamp updates the cache timestamp.
func touchCacheTimestamp(repoDir string) {
	timestampFile := filepath.Join(repoDir, ".gobuddy-cache-timestamp")
	f, err := os.Create(timestampFile)
	if err != nil {
		return
	}
	f.Close()
}

// errorResult creates an error result for the MCP tool.
func errorResult(err error) (*mcp.CallToolResult, StandardsOutput, error) {
	errMsg := fmt.Sprintf("Error: %v", err)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: errMsg},
		},
	}, StandardsOutput{}, err
}
