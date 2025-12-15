package main

import (
	"context"
	"log"

	"github.com/jKiler/gobuddy/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gobuddy",
		Version: "v1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "godoc",
		Description: "Get Go documentation for a package or symbol using 'go doc'",
	}, tools.Godoc)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "standards",
		Description: "Fetch and aggregate coding standards from multiple sources (local files, git repositories). Supports priority-based aggregation and caching.",
	}, tools.Standards)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "review",
		Description: "Review Go code against coding standards. Writes standards to .gobuddy/standards.md and provides file paths for review.",
	}, tools.Review)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
