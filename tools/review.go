package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ReviewInput is the input for the review tool.
type ReviewInput struct {
	Path      string `json:"path" jsonschema:"file path or directory to review"`
	Preset    string `json:"preset,omitempty" jsonschema:"standards preset to review against (e.g., google-go)"`
	Standards string `json:"standards,omitempty" jsonschema:"custom standards text to review against"`
}

// ReviewOutput is the output for the review tool.
type ReviewOutput struct {
	StandardsPath string   `json:"standards_path" jsonschema:"path to the coding standards file"`
	FilePaths     []string `json:"file_paths" jsonschema:"paths to Go files to review"`
}

// Review prepares Go code files for review against coding standards.
func Review(ctx context.Context, req *mcp.CallToolRequest, input ReviewInput) (
	*mcp.CallToolResult,
	ReviewOutput,
	error,
) {
	var standards string
	var err error

	if input.Preset != "" {
		standardsInput := StandardsInput{
			Preset: input.Preset,
		}
		_, output, err := Standards(ctx, nil, standardsInput)
		if err != nil {
			return reviewErrorResult(fmt.Errorf("failed to fetch preset standards: %w", err))
		}
		standards = output.Standards
	} else if input.Standards != "" {
		standards = input.Standards
	} else {
		return reviewErrorResult(fmt.Errorf("either preset or standards must be provided"))
	}

	standardsPath, err := writeStandardsFile(standards)
	if err != nil {
		return reviewErrorResult(fmt.Errorf("failed to write standards file: %w", err))
	}

	filePaths, err := collectGoFiles(input.Path)
	if err != nil {
		return reviewErrorResult(fmt.Errorf("failed to collect Go files: %w", err))
	}

	var message strings.Builder
	message.WriteString("Please review the following Go files against the coding standards.\n\n")
	message.WriteString(fmt.Sprintf("**Coding standards:** %s\n\n", standardsPath))
	message.WriteString("**Files to review:**\n")
	for _, path := range filePaths {
		message.WriteString(fmt.Sprintf("- %s\n", path))
	}
	message.WriteString("\n")
	message.WriteString("Read the standards file and each code file, then provide a detailed review identifying:\n")
	message.WriteString("1. Violations of the standards with file:line references\n")
	message.WriteString("2. Severity (error/warning/info)\n")
	message.WriteString("3. Specific suggestions for fixes\n")
	message.WriteString("4. Positive aspects of the code\n\n")
	message.WriteString("Format your review as markdown with clear sections.")

	output := ReviewOutput{
		StandardsPath: standardsPath,
		FilePaths:     filePaths,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: message.String()},
		},
	}, output, nil
}

// writeStandardsFile writes coding standards to .gobuddy/standards.md.
func writeStandardsFile(standards string) (string, error) {
	gobuddyDir := ".gobuddy"
	if err := os.MkdirAll(gobuddyDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create .gobuddy directory: %w", err)
	}

	standardsPath := filepath.Join(gobuddyDir, "standards.md")
	if err := os.WriteFile(standardsPath, []byte(standards), 0644); err != nil {
		return "", fmt.Errorf("failed to write standards file: %w", err)
	}

	return standardsPath, nil
}

// collectGoFiles collects all Go file paths from a file or directory.
func collectGoFiles(path string) ([]string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	var filePaths []string

	if info.IsDir() {
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(filePath, ".go") && !strings.HasSuffix(filePath, "_test.go") {
				filePaths = append(filePaths, filePath)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if len(filePaths) == 0 {
			return nil, fmt.Errorf("no Go files found in directory")
		}
		return filePaths, nil
	}

	if !strings.HasSuffix(path, ".go") {
		return nil, fmt.Errorf("not a Go file: %s", path)
	}
	return []string{path}, nil
}

// reviewErrorResult creates an error result for the review tool.
func reviewErrorResult(err error) (*mcp.CallToolResult, ReviewOutput, error) {
	errMsg := fmt.Sprintf("Error: %v", err)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: errMsg},
		},
	}, ReviewOutput{}, err
}
