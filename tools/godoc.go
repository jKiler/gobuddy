package tools

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GodocInput is the input for the godoc tool.
type GodocInput struct {
	Package string `json:"package" jsonschema:"the Go package to get documentation for"`
	Symbol  string `json:"symbol,omitempty" jsonschema:"the optional symbol (function, type, method) to get documentation for"`
}

// GodocOutput is the output for the godoc tool.
type GodocOutput struct {
	Documentation string `json:"documentation" jsonschema:"the documentation output from go doc"`
}

// Godoc retrieves Go documentation for a package or symbol using 'go doc'.
func Godoc(ctx context.Context, req *mcp.CallToolRequest, input GodocInput) (
	*mcp.CallToolResult,
	GodocOutput,
	error,
) {
	args := []string{"doc"}
	args = append(args, input.Package)
	if input.Symbol != "" {
		args = append(args, input.Symbol)
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Lookup failures (unknown package or symbol) are expected tool
		// outcomes, not protocol errors, so report them via IsError.
		errMsg := fmt.Sprintf("go doc failed: %v\nOutput: %s", err, string(output))
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: errMsg},
			},
		}, GodocOutput{}, nil
	}

	doc := string(output)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: doc},
		},
	}, GodocOutput{Documentation: doc}, nil
}
