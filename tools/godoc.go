package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/jKiler/gobuddy/internal/run"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GodocInput is the input for the godoc tool.
type GodocInput struct {
	Package    string `json:"package" jsonschema:"the Go package to get documentation for"`
	Symbol     string `json:"symbol,omitempty" jsonschema:"the optional symbol (function, type, method) to get documentation for"`
	WorkingDir string `json:"working_dir,omitempty" jsonschema:"directory of the Go module to resolve documentation in, so dependency versions match that module (defaults to the server's working directory)"`
	Mode       string `json:"mode,omitempty" jsonschema:"output mode: doc (default, package or symbol documentation), all (full documentation for the package), or src (source code of the symbol)"`
}

// GodocOutput is the output for the godoc tool.
type GodocOutput struct {
	Documentation string `json:"documentation" jsonschema:"the documentation output from go doc"`
}

// Godoc retrieves Go documentation for a package or symbol using 'go doc',
// resolving packages in the requested module so dependency versions match.
func Godoc(ctx context.Context, req *mcp.CallToolRequest, input GodocInput) (
	*mcp.CallToolResult,
	GodocOutput,
	error,
) {
	if input.WorkingDir != "" {
		info, err := os.Stat(input.WorkingDir)
		if err != nil || !info.IsDir() {
			return godocError(fmt.Sprintf("working_dir %q is not an existing directory", input.WorkingDir))
		}
	}

	args := []string{"doc"}
	switch input.Mode {
	case "", "doc":
	case "all":
		args = append(args, "-all")
	case "src":
		args = append(args, "-src")
	default:
		return godocError(fmt.Sprintf("unknown mode %q: want doc, all, or src", input.Mode))
	}
	args = append(args, input.Package)
	if input.Symbol != "" {
		args = append(args, input.Symbol)
	}

	output, err := run.Command(ctx, input.WorkingDir, "go", args...)
	if err != nil {
		// Lookup failures (unknown package or symbol) are expected tool
		// outcomes, not protocol errors, so report them via IsError.
		return godocError(fmt.Sprintf("go doc failed: %v\nOutput: %s", err, string(output)))
	}

	doc := string(output)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: doc},
		},
	}, GodocOutput{Documentation: doc}, nil
}

// godocError creates an IsError tool result, keeping the Go error nil so
// expected failures are reported in-band rather than as protocol errors.
func godocError(msg string) (*mcp.CallToolResult, GodocOutput, error) {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}, GodocOutput{}, nil
}
