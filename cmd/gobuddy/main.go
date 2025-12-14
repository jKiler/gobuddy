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
		Name:        "check",
		Description: "Validate Go code against coding standards using LLM-based analysis. Provides detailed feedback on violations, suggestions for fixes, and positive aspects.",
	}, tools.Check)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatal(err)
	}
}
