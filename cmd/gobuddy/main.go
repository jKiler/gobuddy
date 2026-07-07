package main

import (
	"context"
	"log"
	"runtime/debug"

	"github.com/jKiler/gobuddy/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// version reports the module version recorded in the build info, so releases
// track git tags instead of a hardcoded constant.
func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(unknown)"
}

// newServer builds the gobuddy MCP server with its full tool surface,
// separated from main so tests can connect to it over an in-memory
// transport.
func newServer() *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gobuddy",
		Version: version(),
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "godoc",
		Description: "Get Go documentation for a package or symbol using 'go doc'. Set working_dir to the target module so dependency versions resolve correctly; mode selects doc (default), all (full package docs), or src (symbol source).",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Go Documentation",
			ReadOnlyHint: true,
		},
	}, tools.Godoc)

	// gocheck only reads the target module (test binaries write nothing
	// user-visible) and shells out to the local toolchain only.
	closedWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name:        "gocheck",
		Description: "Run gofmt, go vet, and the test suite for a Go module in one call and return a compact structured report: unformatted files, vet issues, and failing tests with truncated output.",
		Annotations: &mcp.ToolAnnotations{
			Title:         "Go Quality Gate",
			ReadOnlyHint:  true,
			OpenWorldHint: &closedWorld,
		},
	}, tools.Gocheck)

	return server
}

func main() {
	if err := newServer().Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
