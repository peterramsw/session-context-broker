package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/broker"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/evidence"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func cmdServeMCP(args []string, reader session.TranscriptReader) {
	exitOnError(runServeMCP(args, os.Stdin, os.Stdout, os.Stderr, parser.DefaultStore(), reader))
}

func runServeMCP(args []string, in io.Reader, out io.Writer, errOut io.Writer, store parser.Store, reader session.TranscriptReader) error {
	fs := flag.NewFlagSet("serve-mcp", flag.ContinueOnError)
	fs.SetOutput(errOut)
	configPath := fs.String("config", "", "path to session-context config.json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg := config.LoadSessionContext()
	if *configPath != "" {
		cfg = config.LoadSessionContextFromPath(*configPath)
	}
	server := mcpServer{svc: broker.New(store, reader, cfg), cfg: cfg}
	return server.serve(in, out)
}

type mcpServer struct {
	svc broker.Service
	cfg config.SessionContextConfig
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      any       `json:"id,omitempty"`
	Result  any       `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s mcpServer) serve(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	for {
		msg, err := readMCPMessage(reader)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		var req rpcRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}
		if req.ID == nil {
			continue
		}
		resp := s.handle(req)
		if err := writeMCPMessage(out, resp); err != nil {
			return err
		}
	}
}

func (s mcpServer) handle(req rpcRequest) rpcResponse {
	result, err := s.dispatch(req.Method, req.Params)
	if err != nil {
		return rpcResponse{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32000, Message: err.Error()}}
	}
	return rpcResponse{JSONRPC: "2.0", ID: req.ID, Result: result}
}

func (s mcpServer) dispatch(method string, params json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		return map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]any{"name": "cc-session", "version": version},
			"capabilities":    map[string]any{"tools": map[string]any{}},
		}, nil
	case "tools/list":
		return map[string]any{"tools": mcpTools()}, nil
	case "tools/call":
		var call struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(params, &call); err != nil {
			return nil, err
		}
		return s.callTool(call.Name, call.Arguments)
	default:
		return map[string]any{}, nil
	}
}

func (s mcpServer) callTool(name string, args map[string]any) (any, error) {
	switch name {
	case "list_sessions":
		refs, err := s.svc.List(argString(args, "provider", providerClaudeCode), argString(args, "project", ""), argInt(args, "limit", 20))
		return toolJSON(refs), err
	case "inspect_session":
		filtered, err := s.svc.ResolveFiltered(argString(args, "session_id", ""), argString(args, "provider", providerAuto))
		return toolJSON(filtered.Metadata), err
	case "filter_session":
		filtered, err := s.svc.ResolveFiltered(argString(args, "session_id", ""), argString(args, "provider", providerAuto))
		return toolText(filtered.FilteredText), err
	case "create_handoff":
		result, err := s.svc.CreateHandoff(context.Background(), argString(args, "session_id", ""), argString(args, "provider", providerAuto), broker.HandoffOptions{
			LLMMode:          argString(args, "llm", "auto"),
			MinFilteredChars: argInt(args, "min_filtered_chars", -1),
			Force:            argBool(args, "force", false),
		})
		return toolJSON(result), err
	case "get_handoff":
		path := evidencePath(s.cfg.StorageRoot, argString(args, "provider", ""), argString(args, "session_id", ""), "handoff.json")
		data, err := os.ReadFile(path)
		return toolText(string(data)), err
	case "search_session":
		matches, err := s.svc.SearchSession(argString(args, "session_id", ""), argString(args, "provider", providerAuto), argString(args, "query", ""))
		return toolJSON(matches), err
	case "expand_evidence":
		result, err := evidence.Store{Root: s.cfg.StorageRoot}.Expand(evidence.ExpandOptions{
			Provider:     argString(args, "provider", ""),
			SessionID:    argString(args, "session_id", ""),
			EvidenceID:   argString(args, "evidence_id", ""),
			AllowedRoots: allowedEvidenceRoots(s.cfg),
			Limit:        argInt(args, "limit", 64*1024),
			Unredacted:   argBool(args, "unredacted", false),
		})
		return toolJSON(result), err
	case "compare_context_size":
		filtered, err := s.svc.ResolveFiltered(argString(args, "session_id", ""), argString(args, "provider", providerAuto))
		if err != nil {
			return nil, err
		}
		return toolJSON(map[string]any{"raw_chars": filtered.Stats.RawChars, "filtered_chars": filtered.Stats.FilteredChars}), nil
	case "verify_workspace":
		report, err := s.svc.VerifyWorkspace(argString(args, "path", ""))
		return toolJSON(report), err
	default:
		return nil, fmt.Errorf("unknown tool %q", name)
	}
}

func mcpTools() []map[string]any {
	names := []string{"list_sessions", "inspect_session", "filter_session", "create_handoff", "get_handoff", "search_session", "expand_evidence", "compare_context_size", "verify_workspace"}
	tools := make([]map[string]any, 0, len(names))
	for _, name := range names {
		tools = append(tools, map[string]any{
			"name":        name,
			"description": "cc-session " + strings.ReplaceAll(name, "_", " "),
			"inputSchema": map[string]any{"type": "object", "additionalProperties": true},
		})
	}
	return tools
}

func toolText(text string) map[string]any {
	return map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}
}

func toolJSON(v any) map[string]any {
	data, _ := json.MarshalIndent(v, "", "  ")
	return toolText(string(data))
}

func readMCPMessage(r *bufio.Reader) ([]byte, error) {
	first, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(first, "Content-Length:") {
		length, _ := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(first, "Content-Length:")))
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return nil, err
			}
			if strings.TrimSpace(line) == "" {
				break
			}
		}
		buf := make([]byte, length)
		_, err := io.ReadFull(r, buf)
		return buf, err
	}
	return []byte(strings.TrimSpace(first)), nil
}

func writeMCPMessage(w io.Writer, resp rpcResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	// MCP stdio transport frames each JSON-RPC message as a single line of JSON
	// terminated by a newline (not LSP-style Content-Length headers). json.Marshal
	// emits compact JSON with no embedded newlines, so one message == one line.
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}

func evidencePath(root, provider, sessionID, name string) string {
	return filepathJoin(root, safePathSegment(provider), safePathSegment(sessionID), name)
}

func filepathJoin(parts ...string) string {
	return strings.Join(parts, string(os.PathSeparator))
}

func safePathSegment(v string) string {
	if v == "" {
		return "_"
	}
	var b strings.Builder
	for _, r := range v {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func argString(args map[string]any, key, def string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return def
}

func argInt(args map[string]any, key string, def int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return def
	}
}

func argBool(args map[string]any, key string, def bool) bool {
	if v, ok := args[key].(bool); ok {
		return v
	}
	return def
}

func framedJSON(v any) string {
	data, _ := json.Marshal(v)
	var b bytes.Buffer
	fmt.Fprintf(&b, "Content-Length: %d\r\n\r\n%s", len(data), data)
	return b.String()
}
