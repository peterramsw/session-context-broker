package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/broker"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
)

func mcpTestServer(t *testing.T) *mcpServer {
	t.Helper()
	cfg := config.LoadSessionContextFromPath(writeMCPConfigFile(t, t.TempDir(), map[string]any{"storage_root": t.TempDir()}))
	return &mcpServer{svc: broker.New(parser.Store{}, testReader, cfg), cfg: cfg}
}

// connectMCP wires a real MCP client to the server over the SDK's in-memory
// transport and returns an initialized session — a genuine client round-trip,
// not a self-framed harness.
func connectMCP(t *testing.T, s *mcpServer) (*mcp.ClientSession, context.Context) {
	t.Helper()
	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	if _, err := s.server().Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs, ctx
}

func TestServeMCP_GivenListTools_ThenAdvertisesTypedSessionTools(t *testing.T) {
	cs, ctx := connectMCP(t, mcpTestServer(t))
	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	byName := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		byName[tool.Name] = tool
	}
	for _, want := range []string{"list_sessions", "inspect_session", "filter_session", "create_handoff", "get_handoff", "search_session", "expand_evidence", "compare_context_size", "verify_workspace"} {
		if byName[want] == nil {
			t.Fatalf("tools/list missing %q; got %v", want, keys(byName))
		}
	}
	// list_sessions must advertise a typed "limit" property so clients forward it.
	schema, _ := json.Marshal(byName["list_sessions"].InputSchema)
	for _, prop := range []string{"limit", "provider", "project"} {
		if !strings.Contains(string(schema), prop) {
			t.Fatalf("list_sessions input schema missing typed property %q: %s", prop, schema)
		}
	}
}

func TestServeMCP_GivenCallTool_ThenReturnsContent(t *testing.T) {
	cs, ctx := connectMCP(t, mcpTestServer(t))
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_sessions",
		Arguments: map[string]any{"provider": "claude_code", "limit": 1},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatalf("expected tool content")
	}
}

func TestServeMCP_GivenThreeClients_ThenCoexist(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan error, 3)
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cs, ctx := connectMCP(t, mcpTestServer(t))
			res, err := cs.ListTools(ctx, nil)
			if err != nil {
				errs <- err
				return
			}
			if len(res.Tools) == 0 {
				errs <- os.ErrInvalid
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent client returned error: %v", err)
		}
	}
}

func keys(m map[string]*mcp.Tool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func writeMCPConfigFile(t *testing.T, dir string, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
