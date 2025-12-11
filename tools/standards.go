package tools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	defaultCacheDuration = 336 * time.Hour // 2 weeks (1 sprint)
)

// StandardsSource represents a source of coding standards.
type StandardsSource struct {
	Type     string   `json:"type" jsonschema:"type of source (local or git)"`
	Location string   `json:"location" jsonschema:"file path for local sources or git URL for git sources"`
	Files    []string `json:"files,omitempty" jsonschema:"specific markdown files to fetch from the source"`
	Branch   string   `json:"branch,omitempty" jsonschema:"git branch or tag to checkout (defaults to main)"`
	Priority int      `json:"priority" jsonschema:"priority of this source (lower number = higher priority)"`
}

// StandardsInput is the input for the fetch_coding_standards tool.
type StandardsInput struct {
	Sources       []StandardsSource `json:"sources,omitempty" jsonschema:"list of sources to fetch standards from (uses default sources if empty)"`
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

	// If no sources specified, use defaults
	if len(input.Sources) == 0 {
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
			Location: "git@github.com:your-org/go-standards.git",
			Files:    []string{"STANDARDS.md", "GO_GUIDELINES.md"},
			Branch:   "main",
			Priority: 10,
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
		// Remove old cache
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
