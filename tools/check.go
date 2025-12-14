package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// CheckInput is the input for the check tool.
type CheckInput struct {
	Path      string `json:"path" jsonschema:"file path or directory to check"`
	Preset    string `json:"preset,omitempty" jsonschema:"standards preset to check against (e.g., google-go)"`
	Standards string `json:"standards,omitempty" jsonschema:"custom standards text to check against"`
}

// CheckOutput is the output for the check tool.
type CheckOutput struct {
	Review   string `json:"review" jsonschema:"LLM code review with violations and suggestions"`
	FilePath string `json:"file_path" jsonschema:"path that was checked"`
}

// Check validates Go code against coding standards using LLM-based analysis.
func Check(ctx context.Context, req *mcp.CallToolRequest, input CheckInput) (
	*mcp.CallToolResult,
	CheckOutput,
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
			return checkErrorResult(fmt.Errorf("failed to fetch preset standards: %w", err))
		}
		standards = output.Standards
	} else if input.Standards != "" {
		standards = input.Standards
	} else {
		return checkErrorResult(fmt.Errorf("either preset or standards must be provided"))
	}

	code, err := readCode(input.Path)
	if err != nil {
		return checkErrorResult(fmt.Errorf("failed to read code: %w", err))
	}

	review, err := reviewCodeWithLLM(ctx, standards, code)
	if err != nil {
		return checkErrorResult(fmt.Errorf("failed to review code: %w", err))
	}

	output := CheckOutput{
		Review:   review,
		FilePath: input.Path,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: review},
		},
	}, output, nil
}

// readCode reads Go code from a file or directory.
func readCode(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		// Read all .go files in directory
		var allCode strings.Builder
		err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(filePath, ".go") && !strings.HasSuffix(filePath, "_test.go") {
				data, err := os.ReadFile(filePath)
				if err != nil {
					return err
				}
				allCode.WriteString(fmt.Sprintf("// File: %s\n", filePath))
				allCode.Write(data)
				allCode.WriteString("\n\n")
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		if allCode.Len() == 0 {
			return "", fmt.Errorf("no Go files found in directory")
		}
		return allCode.String(), nil
	}

	// Read single file
	if !strings.HasSuffix(path, ".go") {
		return "", fmt.Errorf("not a Go file: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// reviewCodeWithLLM uses Claude API to review code against standards.
func reviewCodeWithLLM(ctx context.Context, standards, code string) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	prompt := `You are a Go code reviewer. Review the following code against these coding standards:

<standards>
` + standards + `
</standards>

<code>
` + code + `
</code>

Provide a detailed review identifying:
1. Violations of the standards with file:line references
2. Severity (error/warning/info)
3. Specific suggestions for fixes
4. Positive aspects of the code

Format as markdown with clear sections.`

	message, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_5_20250929,
		MaxTokens: 8192,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to call Claude API: %w", err)
	}

	// Extract text from response
	var review strings.Builder
	review.WriteString("# Code Review\n\n")

	for _, contentBlock := range message.Content {
		// Access the text field from the content block
		if contentBlock.Type == "text" {
			review.WriteString(contentBlock.Text)
		}
	}

	return review.String(), nil
}

// checkErrorResult creates an error result for the check tool.
func checkErrorResult(err error) (*mcp.CallToolResult, CheckOutput, error) {
	errMsg := fmt.Sprintf("Error: %v", err)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: errMsg},
		},
	}, CheckOutput{}, err
}
