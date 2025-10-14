package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()

	// Start the server as a subprocess
	cmd := exec.Command("./bin/gobuddy")
	transport := &mcp.CommandTransport{Command: cmd}

	// Create and connect a client
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "v1.0.0",
	}, nil)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer session.Close()

	fmt.Println("Connected to server successfully!")

	// List available tools
	toolsResp, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		log.Fatal("Failed to list tools:", err)
	}

	fmt.Println("\nAvailable tools:")
	for _, tool := range toolsResp.Tools {
		fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
	}

	// Test godoc with fmt package
	fmt.Println("\n--- Testing godoc tool ---")
	fmt.Println("\nCalling godoc with package='fmt'...")
	godocResult, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "godoc",
		Arguments: map[string]interface{}{
			"package": "fmt",
		},
	})
	if err != nil {
		log.Fatal("Failed to call godoc:", err)
	}

	fmt.Println("Documentation (first 500 chars):")
	for _, content := range godocResult.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			text := textContent.Text
			if len(text) > 500 {
				text = text[:500] + "..."
			}
			fmt.Println(text)
		}
	}

	// Test godoc with fmt.Printf symbol
	fmt.Println("\nCalling godoc with package='fmt', symbol='Printf'...")
	godocResult2, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "godoc",
		Arguments: map[string]interface{}{
			"package": "fmt",
			"symbol":  "Printf",
		},
	})
	if err != nil {
		log.Fatal("Failed to call godoc:", err)
	}

	fmt.Println("Documentation:")
	for _, content := range godocResult2.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			fmt.Println(textContent.Text)
		}
	}

	// Test godoc with external package
	fmt.Println("\nCalling godoc with package='github.com/modelcontextprotocol/go-sdk/mcp'...")
	godocResult3, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "godoc",
		Arguments: map[string]interface{}{
			"package": "github.com/modelcontextprotocol/go-sdk/mcp",
		},
	})
	if err != nil {
		log.Fatal("Failed to call godoc:", err)
	}

	fmt.Println("Documentation (first 500 chars):")
	for _, content := range godocResult3.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			text := textContent.Text
			if len(text) > 500 {
				text = text[:500] + "..."
			}
			fmt.Println(text)
		}
	}

	// Test godoc with external package and symbol
	fmt.Println("\nCalling godoc with package='github.com/modelcontextprotocol/go-sdk/mcp', symbol='AddTool'...")
	godocResult4, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "godoc",
		Arguments: map[string]interface{}{
			"package": "github.com/modelcontextprotocol/go-sdk/mcp",
			"symbol":  "AddTool",
		},
	})
	if err != nil {
		log.Fatal("Failed to call godoc:", err)
	}

	fmt.Println("Documentation:")
	for _, content := range godocResult4.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			fmt.Println(textContent.Text)
		}
	}

	fmt.Println("\nAll tests passed!")
	os.Exit(0)
}
