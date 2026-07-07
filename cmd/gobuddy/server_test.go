package main

import (
	"context"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// connect wires a test client to the gobuddy server over an in-memory
// transport and returns the client session.
func connect(t *testing.T) *mcp.ClientSession {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx := context.Background()
	serverSession, err := newServer().Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Wait() })

	client := mcp.NewClient(&mcp.Implementation{Name: "gobuddy-test", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

// TestServerSurface pins the exact set of tools the server exposes, so
// additions and removals are deliberate changes to this list rather than
// accidents.
func TestServerSurface(t *testing.T) {
	session := connect(t)

	res, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}

	var got []string
	for _, tool := range res.Tools {
		got = append(got, tool.Name)
	}
	sort.Strings(got)

	want := []string{"gocheck", "godoc"}
	if len(got) != len(want) {
		t.Fatalf("tool surface = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tool surface = %v, want %v", got, want)
		}
	}

	for _, tool := range res.Tools {
		if tool.Annotations == nil {
			t.Errorf("tool %q has no annotations", tool.Name)
		}
	}
}

// TestServerGodocCall exercises the godoc tool end-to-end through the MCP
// session, including the IsError path for an unknown package.
func TestServerGodocCall(t *testing.T) {
	session := connect(t)
	ctx := context.Background()

	res, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "godoc",
		Arguments: map[string]any{"package": "fmt", "symbol": "Printf"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected IsError result: %v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected non-empty content")
	}

	res, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "godoc",
		Arguments: map[string]any{"package": "nonexistent/package/xyz"},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected IsError result for unknown package")
	}
}
