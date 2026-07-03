package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/broker"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/evidence"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func cmdServeMCP(args []string, reader session.TranscriptReader) {
	exitOnError(runServeMCP(args, os.Stderr, parser.DefaultStore(), reader))
}

// runServeMCP builds the broker-backed MCP server and runs it over the official
// SDK's stdio transport (newline-delimited JSON-RPC).
func runServeMCP(args []string, errOut io.Writer, store parser.Store, reader session.TranscriptReader) error {
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
	s := &mcpServer{svc: broker.New(store, reader, cfg), cfg: cfg}
	return s.server().Run(context.Background(), &mcp.StdioTransport{})
}

type mcpServer struct {
	svc broker.Service
	cfg config.SessionContextConfig
}

// Typed argument structs. The SDK infers each tool's JSON input schema from
// these (field names + types become properties; the jsonschema tag supplies the
// description), so MCP clients forward every declared argument.
type sessionArgs struct {
	SessionID string `json:"session_id" jsonschema:"session id or unique prefix"`
	Provider  string `json:"provider,omitempty" jsonschema:"session provider: auto, claude_code, codex, antigravity, all"`
}

type listArgs struct {
	Provider string `json:"provider,omitempty" jsonschema:"session provider: auto, claude_code, codex, antigravity, all"`
	Project  string `json:"project,omitempty" jsonschema:"filter by project-name substring"`
	Limit    int    `json:"limit,omitempty" jsonschema:"max results (default 20)"`
}

type handoffArgs struct {
	SessionID        string `json:"session_id" jsonschema:"session id or unique prefix"`
	Provider         string `json:"provider,omitempty" jsonschema:"session provider"`
	LLM              string `json:"llm,omitempty" jsonschema:"local LLM mode: auto, always, never"`
	MinFilteredChars *int   `json:"min_filtered_chars,omitempty" jsonschema:"override the --llm auto threshold"`
	Force            bool   `json:"force,omitempty" jsonschema:"overwrite existing artifacts"`
}

type getHandoffArgs struct {
	SessionID string `json:"session_id" jsonschema:"session id or unique prefix"`
	Provider  string `json:"provider,omitempty" jsonschema:"session provider"`
	Format    string `json:"format,omitempty" jsonschema:"output format: json or markdown"`
}

type searchArgs struct {
	SessionID string `json:"session_id" jsonschema:"session id or unique prefix"`
	Provider  string `json:"provider,omitempty" jsonschema:"session provider"`
	Query     string `json:"query" jsonschema:"case-insensitive substring"`
}

type expandArgs struct {
	SessionID  string `json:"session_id" jsonschema:"session id or unique prefix"`
	Provider   string `json:"provider,omitempty" jsonschema:"session provider"`
	EvidenceID string `json:"evidence_id" jsonschema:"evidence id (evi-...)"`
	Limit      int    `json:"limit,omitempty" jsonschema:"max chars to read"`
	Unredacted bool   `json:"unredacted,omitempty" jsonschema:"return unredacted content"`
}

type workspaceArgs struct {
	Path string `json:"path" jsonschema:"workspace path inside allowed_workspace_roots"`
}

func (s *mcpServer) server() *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: "cc-session", Version: version}, nil)
	mcp.AddTool(srv, &mcp.Tool{Name: "list_sessions", Description: "List sessions for a provider"}, s.listSessions)
	mcp.AddTool(srv, &mcp.Tool{Name: "inspect_session", Description: "Show session metadata and message/tool counts"}, s.inspectSession)
	mcp.AddTool(srv, &mcp.Tool{Name: "filter_session", Description: "Return the deterministic filtered transcript"}, s.filterSession)
	mcp.AddTool(srv, &mcp.Tool{Name: "create_handoff", Description: "Write filtered/evidence artifacts and optionally a local-LLM handoff"}, s.createHandoff)
	mcp.AddTool(srv, &mcp.Tool{Name: "get_handoff", Description: "Read a previously written handoff"}, s.getHandoff)
	mcp.AddTool(srv, &mcp.Tool{Name: "search_session", Description: "Search evidence summaries"}, s.searchSession)
	mcp.AddTool(srv, &mcp.Tool{Name: "expand_evidence", Description: "Expand one evidence id to its source content"}, s.expandEvidence)
	mcp.AddTool(srv, &mcp.Tool{Name: "compare_context_size", Description: "Compare raw vs filtered character size"}, s.compareContextSize)
	mcp.AddTool(srv, &mcp.Tool{Name: "verify_workspace", Description: "Read-only git verification inside allowed_workspace_roots"}, s.verifyWorkspace)
	return srv
}

func (s *mcpServer) listSessions(_ context.Context, _ *mcp.CallToolRequest, in listArgs) (*mcp.CallToolResult, any, error) {
	provider := in.Provider
	if provider == "" {
		provider = providerClaudeCode
	}
	limit := in.Limit
	if limit == 0 {
		limit = 20
	}
	refs, err := s.svc.List(provider, in.Project, limit)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(refs), nil, nil
}

func (s *mcpServer) inspectSession(_ context.Context, _ *mcp.CallToolRequest, in sessionArgs) (*mcp.CallToolResult, any, error) {
	filtered, err := s.svc.ResolveFiltered(in.SessionID, providerOrAuto(in.Provider))
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(filtered.Metadata), nil, nil
}

func (s *mcpServer) filterSession(_ context.Context, _ *mcp.CallToolRequest, in sessionArgs) (*mcp.CallToolResult, any, error) {
	filtered, err := s.svc.ResolveFiltered(in.SessionID, providerOrAuto(in.Provider))
	if err != nil {
		return nil, nil, err
	}
	return textResult(filtered.FilteredText), nil, nil
}

func (s *mcpServer) createHandoff(ctx context.Context, _ *mcp.CallToolRequest, in handoffArgs) (*mcp.CallToolResult, any, error) {
	minChars := -1
	if in.MinFilteredChars != nil {
		minChars = *in.MinFilteredChars
	}
	llm := in.LLM
	if llm == "" {
		llm = "auto"
	}
	result, err := s.svc.CreateHandoff(ctx, in.SessionID, providerOrAuto(in.Provider), broker.HandoffOptions{
		LLMMode:          llm,
		MinFilteredChars: minChars,
		Force:            in.Force,
	})
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(result), nil, nil
}

func (s *mcpServer) getHandoff(_ context.Context, _ *mcp.CallToolRequest, in getHandoffArgs) (*mcp.CallToolResult, any, error) {
	prov, sid, err := resolveStoredSession(s.cfg.StorageRoot, in.Provider, in.SessionID)
	if err != nil {
		return nil, nil, err
	}
	name := "handoff.json"
	if in.Format == "markdown" {
		name = "handoff.md"
	}
	data, err := os.ReadFile(evidencePath(s.cfg.StorageRoot, prov, sid, name))
	if err != nil {
		return nil, nil, err
	}
	return textResult(string(data)), nil, nil
}

func (s *mcpServer) searchSession(_ context.Context, _ *mcp.CallToolRequest, in searchArgs) (*mcp.CallToolResult, any, error) {
	matches, err := s.svc.SearchSession(in.SessionID, providerOrAuto(in.Provider), in.Query)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(matches), nil, nil
}

func (s *mcpServer) expandEvidence(_ context.Context, _ *mcp.CallToolRequest, in expandArgs) (*mcp.CallToolResult, any, error) {
	prov, sid, err := resolveStoredSession(s.cfg.StorageRoot, in.Provider, in.SessionID)
	if err != nil {
		return nil, nil, err
	}
	limit := in.Limit
	if limit == 0 {
		limit = 64 * 1024
	}
	result, err := evidence.Store{Root: s.cfg.StorageRoot}.Expand(evidence.ExpandOptions{
		Provider:     prov,
		SessionID:    sid,
		EvidenceID:   in.EvidenceID,
		AllowedRoots: allowedEvidenceRoots(s.cfg),
		Limit:        limit,
		Unredacted:   in.Unredacted,
	})
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(result), nil, nil
}

func (s *mcpServer) compareContextSize(_ context.Context, _ *mcp.CallToolRequest, in sessionArgs) (*mcp.CallToolResult, any, error) {
	filtered, err := s.svc.ResolveFiltered(in.SessionID, providerOrAuto(in.Provider))
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(map[string]any{"raw_chars": filtered.Stats.RawChars, "filtered_chars": filtered.Stats.FilteredChars}), nil, nil
}

func (s *mcpServer) verifyWorkspace(_ context.Context, _ *mcp.CallToolRequest, in workspaceArgs) (*mcp.CallToolResult, any, error) {
	report, err := s.svc.VerifyWorkspace(in.Path)
	if err != nil {
		return nil, nil, err
	}
	return jsonResult(report), nil, nil
}

func providerOrAuto(provider string) string {
	if provider == "" {
		return providerAuto
	}
	return provider
}

func jsonResult(v any) *mcp.CallToolResult {
	data, _ := json.MarshalIndent(v, "", "  ")
	return textResult(string(data))
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func evidencePath(root, provider, sessionID, name string) string {
	return filepathJoin(root, safePathSegment(provider), safePathSegment(sessionID), name)
}

// resolveStoredSession maps a session-id prefix to the (provider, full session id)
// of a persisted evidence-store directory, so get_handoff/expand_evidence accept
// prefixes like the other tools instead of requiring the full session id.
func resolveStoredSession(root, provider, prefix string) (string, string, error) {
	if strings.TrimSpace(prefix) == "" {
		return "", "", fmt.Errorf("session_id is required")
	}
	var providers []string
	switch broker.NormalizeProvider(provider) {
	case session.ProviderClaudeCode, session.ProviderCodex, session.ProviderAntigravity:
		providers = []string{broker.NormalizeProvider(provider)}
	default: // auto, all, or empty: search every provider
		providers = []string{session.ProviderClaudeCode, session.ProviderCodex, session.ProviderAntigravity}
	}
	for _, prov := range providers {
		entries, err := os.ReadDir(filepathJoin(root, safePathSegment(prov)))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() && (e.Name() == prefix || strings.HasPrefix(e.Name(), prefix)) {
				return prov, e.Name(), nil
			}
		}
	}
	return "", "", fmt.Errorf("no stored session matching %q (create a handoff first)", prefix)
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
