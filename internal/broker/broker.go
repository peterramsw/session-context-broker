// Package broker contains the shared session operations used by CLI commands
// and the MCP surface.
package broker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/analyzer"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/antigravitycodec"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/claudecodec"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/codexcodec"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/distiller"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/evidence"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/handoff"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/redaction"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

const (
	ProviderAuto = "auto"
	ProviderAll  = "all"
)

type Service struct {
	Store  parser.Store
	Reader session.TranscriptReader
	Config config.SessionContextConfig
}

type FilteredSession struct {
	Info         handoff.SessionInfo
	Events       []session.SessionEvent
	LegacyEvents []session.Event
	FilteredText string
	Stats        analyzer.StatsResult
	Metadata     session.SessionMetadata
}

type HandoffOptions struct {
	LLMMode          string
	MinFilteredChars int
	Force            bool
}

type HandoffResult struct {
	Mode               string                `json:"mode"`
	Provider           string                `json:"provider"`
	SessionID          string                `json:"session_id"`
	LLMDecision        string                `json:"llm_decision"`
	LLMThreshold       int                   `json:"llm_threshold"`
	RawChars           int                   `json:"raw_chars"`
	FilteredChars      int                   `json:"filtered_chars"`
	RedactedInputChars int                   `json:"redacted_input_chars"`
	Model              string                `json:"model,omitempty"`
	MaxContext         int                   `json:"max_context,omitempty"`
	MaxOutputTokens    int                   `json:"max_output_tokens,omitempty"`
	Temperature        float64               `json:"temperature,omitempty"`
	TopP               *float64              `json:"top_p,omitempty"`
	TopK               int                   `json:"top_k,omitempty"`
	FilteredPath       string                `json:"filtered_path"`
	OutputDir          string                `json:"output_dir,omitempty"`
	EvidenceIndexPath  string                `json:"evidence_index_path"`
	Handoff            *handoff.Handoff      `json:"handoff,omitempty"`
	Diagnostics        distiller.Diagnostics `json:"diagnostics,omitempty"`
}

type WorkspaceReport struct {
	Path   string `json:"path"`
	Branch string `json:"branch"`
	Commit string `json:"commit"`
	Status string `json:"status"`
}

func New(store parser.Store, reader session.TranscriptReader, cfg config.SessionContextConfig) Service {
	if reader == nil {
		reader = claudecodec.Codec{}
	}
	if store.ProjectsDir == "" && store.SessionMetaDir == "" {
		store = parser.DefaultStore()
	}
	// list_sessions falls back to scanning JSONL transcript headers when
	// session-meta is sparse (the common case), which requires a HeaderScanner.
	// The CLI injects one via DefaultStoreWith; the MCP/broker path did not, so
	// list_sessions returned an empty result. Reuse the reader when it can scan
	// headers, otherwise fall back to the claudecodec scanner.
	if store.HeaderScanner == nil {
		if hs, ok := reader.(session.HeaderScanner); ok {
			store.HeaderScanner = hs
		} else {
			store.HeaderScanner = claudecodec.Codec{}
		}
	}
	return Service{Store: store, Reader: reader, Config: cfg}
}

func NormalizeProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "claude", "claude-code", "claude_code":
		return session.ProviderClaudeCode
	case "codex":
		return session.ProviderCodex
	case "antigravity", "angravity":
		return session.ProviderAntigravity
	case "all":
		return ProviderAll
	case "auto":
		return ProviderAuto
	default:
		return strings.ToLower(strings.TrimSpace(provider))
	}
}

func (s Service) List(provider, project string, limit int) ([]session.SessionRef, error) {
	switch NormalizeProvider(provider) {
	case session.ProviderCodex:
		return listProvider(codexcodec.Codec{Roots: s.sourceRoots(session.ProviderCodex)}, project, limit)
	case session.ProviderAntigravity:
		return listProvider(antigravitycodec.Codec{Roots: s.sourceRoots(session.ProviderAntigravity)}, project, limit)
	case session.ProviderClaudeCode:
		store := s.claudeStore()
		entries, warnings := store.ListAllSessions()
		if len(warnings) > 0 {
			return nil, fmt.Errorf(strings.Join(warnings, "; "))
		}
		var refs []session.SessionRef
		project = strings.ToLower(project)
		for _, entry := range entries {
			if limit > 0 && len(refs) >= limit {
				break
			}
			if project != "" && !strings.Contains(strings.ToLower(filepath.Base(entry.ProjectPath)), project) {
				continue
			}
			refs = append(refs, session.SessionRef{
				ID:          entry.SessionID,
				Provider:    session.ProviderClaudeCode,
				ProjectPath: entry.ProjectPath,
				StartTime:   entry.StartTime,
				FirstPrompt: entry.FirstPrompt,
			})
		}
		return refs, nil
	default:
		return nil, fmt.Errorf("unsupported provider for list: %s", provider)
	}
}

func listProvider(provider session.SessionProvider, project string, limit int) ([]session.SessionRef, error) {
	refs, err := provider.Discover()
	if err != nil {
		return nil, err
	}
	project = strings.ToLower(project)
	out := refs[:0:0]
	for _, ref := range refs {
		if limit > 0 && len(out) >= limit {
			break
		}
		if project != "" && !strings.Contains(strings.ToLower(filepath.Base(ref.ProjectPath)), project) {
			continue
		}
		out = append(out, ref)
	}
	return out, nil
}

func (s Service) ResolveFiltered(prefix, provider string) (FilteredSession, error) {
	switch NormalizeProvider(provider) {
	case ProviderAuto:
		if got, err := s.ResolveFiltered(prefix, session.ProviderCodex); err == nil {
			return got, nil
		}
		if got, err := s.ResolveFiltered(prefix, session.ProviderAntigravity); err == nil {
			return got, nil
		}
		return s.ResolveFiltered(prefix, session.ProviderClaudeCode)
	case session.ProviderCodex:
		codec := codexcodec.Codec{Roots: s.sourceRoots(session.ProviderCodex)}
		ref, err := codec.Resolve(prefix)
		if err != nil {
			return FilteredSession{}, err
		}
		return buildProviderFiltered(codec, ref)
	case session.ProviderAntigravity:
		codec := antigravitycodec.Codec{Roots: s.sourceRoots(session.ProviderAntigravity)}
		ref, err := codec.Resolve(prefix)
		if err != nil {
			return FilteredSession{}, err
		}
		return buildProviderFiltered(codec, ref)
	case session.ProviderClaudeCode:
		return s.resolveClaudeFiltered(prefix)
	default:
		return FilteredSession{}, fmt.Errorf("unknown provider %q", provider)
	}
}

func buildProviderFiltered(provider session.SessionProvider, ref session.SessionRef) (FilteredSession, error) {
	events, parseErrors, err := provider.Parse(ref)
	if err != nil {
		return FilteredSession{}, err
	}
	meta, _ := provider.Inspect(ref)
	meta.ParseErrors = append(meta.ParseErrors, parseErrors...)
	legacyReader, ok := provider.(interface {
		ReadAll(string) ([]session.Event, error)
	})
	if !ok {
		return FilteredSession{}, fmt.Errorf("provider %s cannot render legacy filtered text", provider.Name())
	}
	legacy, err := legacyReader.ReadAll(ref.Path)
	if err != nil {
		return FilteredSession{}, err
	}
	stats := analyzer.ComputeStats(legacy)
	workspace := meta.CWD
	if workspace == "" {
		workspace = ref.ProjectPath
	}
	return FilteredSession{
		Info: handoff.SessionInfo{
			Provider:      provider.Name(),
			SessionID:     ref.ID,
			SourcePath:    ref.Path,
			Workspace:     workspace,
			Model:         meta.ModelProvider,
			RawChars:      stats.RawChars,
			FilteredChars: stats.FilteredChars,
		},
		Events:       events,
		LegacyEvents: legacy,
		FilteredText: stats.FilteredText,
		Stats:        stats,
		Metadata:     meta,
	}, nil
}

func (s Service) resolveClaudeFiltered(prefix string) (FilteredSession, error) {
	resolved, err := s.claudeStore().ResolveSession(prefix)
	if err != nil {
		return FilteredSession{}, err
	}
	if resolved.Path == "" {
		return FilteredSession{}, fmt.Errorf("transcript not found: %s", resolved.ID)
	}
	legacy, err := s.Reader.ReadAll(resolved.Path)
	if err != nil {
		return FilteredSession{}, err
	}
	stats := analyzer.ComputeStats(legacy)
	events := legacyToNormalized(session.ProviderClaudeCode, resolved.ID, resolved.Path, legacy)
	return FilteredSession{
		Info: handoff.SessionInfo{
			Provider:      session.ProviderClaudeCode,
			SessionID:     resolved.ID,
			SourcePath:    resolved.Path,
			Workspace:     filepath.Base(filepath.Dir(resolved.Path)),
			RawChars:      stats.RawChars,
			FilteredChars: stats.FilteredChars,
		},
		Events:       events,
		LegacyEvents: legacy,
		FilteredText: stats.FilteredText,
		Stats:        stats,
		Metadata:     session.SessionMetadata{Ref: session.SessionRef{ID: resolved.ID, Provider: session.ProviderClaudeCode, Path: resolved.Path}},
	}, nil
}

func (s Service) sourceRoots(provider string) []string {
	if source, ok := s.Config.SessionSources[provider]; ok {
		return source.Roots
	}
	return nil
}

func (s Service) claudeStore() parser.Store {
	store := s.Store
	if roots := s.sourceRoots(session.ProviderClaudeCode); len(roots) > 0 {
		store.ProjectsDir = roots[0]
	}
	return store
}

func (s Service) CreateHandoff(ctx context.Context, prefix, provider string, opts HandoffOptions) (HandoffResult, error) {
	filtered, err := s.ResolveFiltered(prefix, provider)
	if err != nil {
		return HandoffResult{}, err
	}
	redactedFiltered := redaction.RedactSecrets(filtered.FilteredText)
	minChars := opts.MinFilteredChars
	if minChars < 0 {
		minChars = s.Config.LocalLLM.MinFilteredCharsOrDefault()
	}
	mode := strings.ToLower(strings.TrimSpace(opts.LLMMode))
	if mode == "" {
		mode = "auto"
	}
	useLLM, reason, err := decideLLM(mode, len(redactedFiltered), minChars, s.Config.LocalLLM.IsEnabled())
	if err != nil {
		return HandoffResult{}, err
	}
	evStore := evidence.Store{Root: s.Config.StorageRoot}
	if !useLLM {
		wr, err := evStore.Write(evidence.WriteInput{
			Session:      filtered.Info,
			Events:       filtered.Events,
			FilteredText: redactedFiltered,
			Force:        opts.Force,
		})
		if err != nil {
			return HandoffResult{}, err
		}
		return HandoffResult{
			Mode:               "filtered",
			Provider:           filtered.Info.Provider,
			SessionID:          filtered.Info.SessionID,
			LLMDecision:        reason,
			LLMThreshold:       minChars,
			RawChars:           filtered.Info.RawChars,
			FilteredChars:      filtered.Info.FilteredChars,
			RedactedInputChars: len(redactedFiltered),
			FilteredPath:       wr.FilteredMarkdownPath,
			EvidenceIndexPath:  wr.IndexPath,
		}, nil
	}
	req := distiller.Request{Config: s.Config.LocalLLM, Session: filtered.Info, FilteredTranscript: redactedFiltered}
	generated, diag, err := distiller.Generate(ctx, req, distiller.NewClient(s.Config.LocalLLM))
	if err != nil {
		return HandoffResult{}, err
	}
	evidenceMap := map[string]bool{}
	for _, entry := range evidence.BuildIndex(filtered.Info, filtered.Events).Entries {
		evidenceMap[entry.EvidenceID] = true
	}
	generated = handoff.NormalizeAndValidate(generated, evidenceMap)
	wr, err := evStore.Write(evidence.WriteInput{
		Session:      filtered.Info,
		Events:       filtered.Events,
		FilteredText: redactedFiltered,
		Handoff:      &generated,
		Force:        opts.Force,
	})
	if err != nil {
		return HandoffResult{}, err
	}
	return HandoffResult{
		Mode:               "llm",
		Provider:           filtered.Info.Provider,
		SessionID:          filtered.Info.SessionID,
		LLMDecision:        reason,
		LLMThreshold:       minChars,
		RawChars:           filtered.Info.RawChars,
		FilteredChars:      filtered.Info.FilteredChars,
		RedactedInputChars: diag.RedactedInputChars,
		Model:              s.Config.LocalLLM.Model,
		MaxContext:         s.Config.LocalLLM.MaxContext,
		MaxOutputTokens:    s.Config.LocalLLM.MaxOutputTokens,
		Temperature:        temperatureOrDefault(s.Config.LocalLLM.Temperature),
		TopP:               s.Config.LocalLLM.TopP,
		TopK:               s.Config.LocalLLM.TopK,
		FilteredPath:       wr.FilteredMarkdownPath,
		OutputDir:          wr.Dir,
		EvidenceIndexPath:  wr.IndexPath,
		Handoff:            &generated,
		Diagnostics:        diag,
	}, nil
}

func temperatureOrDefault(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func (s Service) SearchSession(prefix, provider, query string) ([]evidence.Entry, error) {
	filtered, err := s.ResolveFiltered(prefix, provider)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(query)
	var matches []evidence.Entry
	for _, entry := range evidence.BuildIndex(filtered.Info, filtered.Events).Entries {
		if strings.Contains(strings.ToLower(entry.Summary), query) || strings.Contains(strings.ToLower(entry.EventType), query) || strings.Contains(strings.ToLower(entry.ToolName), query) {
			matches = append(matches, entry)
		}
	}
	return matches, nil
}

func (s Service) VerifyWorkspace(path string) (WorkspaceReport, error) {
	resolved, err := resolveAllowedWorkspace(path, s.Config.AllowedWorkspaceRoot)
	if err != nil {
		return WorkspaceReport{}, err
	}
	branch, _ := gitOutput(resolved, "branch", "--show-current")
	commit, _ := gitOutput(resolved, "rev-parse", "HEAD")
	status, err := gitOutput(resolved, "status", "--short", "--branch")
	if err != nil {
		return WorkspaceReport{}, err
	}
	return WorkspaceReport{Path: resolved, Branch: strings.TrimSpace(branch), Commit: strings.TrimSpace(commit), Status: strings.TrimSpace(status)}, nil
}

func decideLLM(mode string, chars, threshold int, enabled bool) (bool, string, error) {
	switch mode {
	case "never":
		return false, "--llm never requested", nil
	case "always":
		if !enabled {
			return false, "--llm always requested, but Local LLM is not enabled", nil
		}
		return true, "--llm always requested", nil
	case "auto":
		if chars < threshold {
			return false, fmt.Sprintf("redacted filtered chars %d below threshold %d", chars, threshold), nil
		}
		if !enabled {
			return false, fmt.Sprintf("redacted filtered chars %d meets threshold %d, but Local LLM is not enabled", chars, threshold), nil
		}
		return true, fmt.Sprintf("redacted filtered chars %d meets threshold %d", chars, threshold), nil
	default:
		return false, "", fmt.Errorf("llm mode must be auto, always, or never")
	}
}

func resolveAllowedWorkspace(path string, roots []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("workspace path is required")
	}
	abs, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = eval
	}
	for _, root := range roots {
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(filepath.Clean(root))
		if err != nil {
			continue
		}
		if evalRoot, err := filepath.EvalSymlinks(absRoot); err == nil {
			absRoot = evalRoot
		}
		if isWithin(absRoot, abs) {
			return abs, nil
		}
	}
	return "", fmt.Errorf("workspace is outside allowed_workspace_roots")
}

func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, string(out))
	}
	return string(out), nil
}

func legacyToNormalized(provider, sessionID, path string, events []session.Event) []session.SessionEvent {
	out := make([]session.SessionEvent, 0, len(events))
	for i, event := range events {
		ne := session.SessionEvent{
			SessionID: sessionID,
			Provider:  provider,
			Timestamp: event.Timestamp,
			Sequence:  i + 1,
			Source:    session.EventSource{Path: path, LineStart: i + 1, LineEnd: i + 1},
			Metadata:  map[string]any{"legacy_raw_type": event.RawType},
		}
		switch event.Kind {
		case session.EventUserMessage:
			ne.EventType = "message"
			ne.Role = "user"
			if event.User != nil {
				ne.Content = event.User.Text
			}
		case session.EventAssistantMessage:
			ne.EventType = "message"
			ne.Role = "assistant"
			if event.Assistant != nil {
				ne.Content = event.Assistant.Text
				if len(event.Assistant.ToolUses) > 0 {
					tool := event.Assistant.ToolUses[0]
					ne.EventType = "tool_call"
					ne.Tool = &session.SessionTool{CallID: tool.ID, Name: tool.Name, Arguments: tool.Input.MarshalNoEscape(), Status: "unknown"}
					ne.Content = ne.Tool.Arguments
				}
			}
		case session.EventToolResult:
			ne.EventType = "tool_result"
			if event.Tool != nil {
				status := "ok"
				if !event.Tool.Success {
					status = "FAILED"
				}
				ne.Tool = &session.SessionTool{CallID: event.Tool.ToolUseID, Name: event.Tool.RawName, Result: event.Tool.Text, Status: status}
				ne.Content = event.Tool.Text
			}
		default:
			ne.EventType = string(event.Kind)
			if event.Noise != nil {
				ne.Content = event.Noise.Text
			}
		}
		ne.EventID = session.StableEventID(sessionID, provider, ne.Sequence, ne.Source, ne.EventType)
		out = append(out, ne)
	}
	return out
}

func isWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
