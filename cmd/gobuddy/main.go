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

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gobuddy",
		Version: version(),
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "godoc",
		Description: "Get Go documentation for a package or symbol using 'go doc'",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Go Documentation",
			ReadOnlyHint: true,
		},
	}, tools.Godoc)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "standards",
		Description: "Fetch and aggregate coding standards from multiple sources (local files, git repositories). Supports priority-based aggregation and caching.",
		Annotations: &mcp.ToolAnnotations{
			Title:        "Coding Standards",
			ReadOnlyHint: true,
		},
	}, tools.Standards)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
