// Package evidence persists derived, redacted session artifacts and maps
// compact evidence IDs back to bounded source-byte ranges.
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/handoff"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/redaction"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

const (
	ManifestFile   = "manifest.json"
	NormalizedFile = "normalized.jsonl"
	FilteredFile   = "filtered.jsonl"
	IndexFile      = "evidence-index.json"
	defaultLimit   = 64 * 1024
)

type Manifest struct {
	SchemaVersion string `json:"schema_version"`
	Provider      string `json:"provider"`
	SessionID     string `json:"session_id"`
	SourcePath    string `json:"source_path"`
	Workspace     string `json:"workspace"`
	RawChars      int    `json:"raw_chars"`
	FilteredChars int    `json:"filtered_chars"`
	GeneratedAt   string `json:"generated_at"`
}

type Index struct {
	SchemaVersion string  `json:"schema_version"`
	Provider      string  `json:"provider"`
	SessionID     string  `json:"session_id"`
	Entries       []Entry `json:"entries"`
}

type Entry struct {
	EvidenceID    string              `json:"evidence_id"`
	EventID       string              `json:"event_id"`
	EventType     string              `json:"event_type"`
	Role          string              `json:"role,omitempty"`
	ToolName      string              `json:"tool_name,omitempty"`
	ToolCallID    string              `json:"tool_call_id,omitempty"`
	Summary       string              `json:"summary"`
	Status        string              `json:"status,omitempty"`
	ExitCode      int                 `json:"exit_code,omitempty"`
	HasExitCode   bool                `json:"has_exit_code,omitempty"`
	RawChars      int                 `json:"raw_chars"`
	FilteredChars int                 `json:"filtered_chars"`
	Source        session.EventSource `json:"source"`
}

type Store struct {
	Root string
}

type WriteInput struct {
	Session      handoff.SessionInfo
	Events       []session.SessionEvent
	FilteredText string
	Handoff      *handoff.Handoff
	Force        bool
}

type WriteResult struct {
	Dir                  string
	ManifestPath         string
	NormalizedPath       string
	FilteredPath         string
	FilteredMarkdownPath string
	IndexPath            string
	HandoffJSONPath      string
	HandoffMarkdownPath  string
	Index                Index
}

type ExpandOptions struct {
	Provider     string
	SessionID    string
	EvidenceID   string
	AllowedRoots []string
	Limit        int
	Unredacted   bool
}

type ExpandResult struct {
	EvidenceID string              `json:"evidence_id"`
	Content    string              `json:"content"`
	Redacted   bool                `json:"redacted"`
	Truncated  bool                `json:"truncated"`
	BytesRead  int                 `json:"bytes_read"`
	Source     session.EventSource `json:"source"`
}

func (s Store) Write(input WriteInput) (WriteResult, error) {
	if s.Root == "" {
		return WriteResult{}, fmt.Errorf("storage root is required")
	}
	if input.Session.Provider == "" || input.Session.SessionID == "" {
		return WriteResult{}, fmt.Errorf("session provider and session_id are required")
	}
	dir, err := s.sessionDir(input.Session.Provider, input.Session.SessionID)
	if err != nil {
		return WriteResult{}, err
	}
	unlock, err := acquireLock(dir)
	if err != nil {
		return WriteResult{}, err
	}
	defer unlock()

	if !input.Force {
		if _, err := os.Stat(filepath.Join(dir, ManifestFile)); err == nil {
			return WriteResult{}, fmt.Errorf("evidence artifacts already exist at %s; rerun with --force to overwrite", dir)
		}
	}

	idx := BuildIndex(input.Session, input.Events)
	manifest := Manifest{
		SchemaVersion: "session-context-evidence/v1",
		Provider:      input.Session.Provider,
		SessionID:     input.Session.SessionID,
		SourcePath:    input.Session.SourcePath,
		Workspace:     input.Session.Workspace,
		RawChars:      input.Session.RawChars,
		FilteredChars: input.Session.FilteredChars,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	manifestPath := filepath.Join(dir, ManifestFile)
	normalizedPath := filepath.Join(dir, NormalizedFile)
	filteredPath := filepath.Join(dir, FilteredFile)
	filteredMarkdownPath := filepath.Join(dir, "filtered.md")
	indexPath := filepath.Join(dir, IndexFile)
	if err := writeJSON(manifestPath, manifest); err != nil {
		return WriteResult{}, err
	}
	if err := writeJSONL(normalizedPath, input.Events); err != nil {
		return WriteResult{}, err
	}
	if err := writeJSONL(filteredPath, []map[string]any{{
		"schema_version": "session-context-filtered/v1",
		"provider":       input.Session.Provider,
		"session_id":     input.Session.SessionID,
		"text":           redaction.RedactSecrets(input.FilteredText),
	}}); err != nil {
		return WriteResult{}, err
	}
	if err := atomicWrite(filteredMarkdownPath, []byte(renderFilteredMarkdown(input.Session, redaction.RedactSecrets(input.FilteredText)))); err != nil {
		return WriteResult{}, err
	}
	if err := writeJSON(indexPath, idx); err != nil {
		return WriteResult{}, err
	}
	result := WriteResult{
		Dir:                  dir,
		ManifestPath:         manifestPath,
		NormalizedPath:       normalizedPath,
		FilteredPath:         filteredPath,
		FilteredMarkdownPath: filteredMarkdownPath,
		IndexPath:            indexPath,
		Index:                idx,
	}
	if input.Handoff != nil {
		if err := writeJSON(filepath.Join(dir, "handoff.json"), input.Handoff); err != nil {
			return WriteResult{}, err
		}
		if err := atomicWrite(filepath.Join(dir, "handoff.md"), []byte(handoff.RenderMarkdown(*input.Handoff))); err != nil {
			return WriteResult{}, err
		}
		result.HandoffJSONPath = filepath.Join(dir, "handoff.json")
		result.HandoffMarkdownPath = filepath.Join(dir, "handoff.md")
	}
	return result, nil
}

func renderFilteredMarkdown(info handoff.SessionInfo, filtered string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Filtered Session\n\n")
	fmt.Fprintf(&b, "Provider: %s\n\n", info.Provider)
	fmt.Fprintf(&b, "Session: %s\n\n", info.SessionID)
	if info.SourcePath != "" {
		fmt.Fprintf(&b, "Source: %s\n\n", info.SourcePath)
	}
	if info.Workspace != "" {
		fmt.Fprintf(&b, "Workspace: %s\n\n", info.Workspace)
	}
	fmt.Fprintf(&b, "Raw chars: %d\n\nFiltered chars: %d\n\n", info.RawChars, info.FilteredChars)
	b.WriteString(filtered)
	if !strings.HasSuffix(filtered, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func BuildIndex(info handoff.SessionInfo, events []session.SessionEvent) Index {
	idx := Index{
		SchemaVersion: "session-context-evidence-index/v1",
		Provider:      info.Provider,
		SessionID:     info.SessionID,
		Entries:       []Entry{},
	}
	for _, event := range events {
		if !isEvidenceWorthy(event) {
			continue
		}
		entry := Entry{
			EvidenceID:    StableEvidenceID(event),
			EventID:       event.EventID,
			EventType:     event.EventType,
			Role:          event.Role,
			Summary:       summarizeEvent(event),
			RawChars:      utf8.RuneCountInString(event.Content),
			FilteredChars: utf8.RuneCountInString(summarizeEvent(event)),
			Source:        event.Source,
		}
		if event.Tool != nil {
			entry.ToolName = event.Tool.Name
			entry.ToolCallID = event.Tool.CallID
			entry.Status = event.Tool.Status
			entry.ExitCode = event.Tool.ExitCode
			entry.HasExitCode = event.Tool.HasExitCode
			if event.Tool.Result != "" {
				entry.RawChars = utf8.RuneCountInString(event.Tool.Result)
			} else if event.Tool.Arguments != "" {
				entry.RawChars = utf8.RuneCountInString(event.Tool.Arguments)
			}
		}
		idx.Entries = append(idx.Entries, entry)
	}
	return idx
}

func StableEvidenceID(event session.SessionEvent) string {
	key := event.EventID
	if key == "" {
		key = fmt.Sprintf("%s|%s|%d|%s|%d|%d|%s", event.Provider, event.SessionID, event.Sequence, event.Source.Path, event.Source.LineStart, event.Source.ByteStart, event.EventType)
	}
	sum := sha256.Sum256([]byte(key))
	return "evi-" + hex.EncodeToString(sum[:])[:16]
}

func (s Store) Expand(opts ExpandOptions) (ExpandResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = defaultLimit
	}
	idx, err := s.ReadIndex(opts.Provider, opts.SessionID)
	if err != nil {
		return ExpandResult{}, err
	}
	var entry *Entry
	for i := range idx.Entries {
		if idx.Entries[i].EvidenceID == opts.EvidenceID {
			entry = &idx.Entries[i]
			break
		}
	}
	if entry == nil {
		return ExpandResult{}, fmt.Errorf("evidence id not found: %s", opts.EvidenceID)
	}
	sourcePath, err := resolveAllowedSource(entry.Source.Path, opts.AllowedRoots)
	if err != nil {
		return ExpandResult{}, err
	}
	data, err := readSourceRange(sourcePath, entry.Source.ByteStart, entry.Source.ByteEnd, opts.Limit)
	if err != nil {
		return ExpandResult{}, err
	}
	truncated := false
	if int64(len(data)) < entry.Source.ByteEnd-entry.Source.ByteStart {
		truncated = true
	}
	text := string(data)
	redacted := false
	if !opts.Unredacted {
		text = redaction.RedactSecrets(text)
		redacted = true
	}
	return ExpandResult{
		EvidenceID: opts.EvidenceID,
		Content:    text,
		Redacted:   redacted,
		Truncated:  truncated,
		BytesRead:  len(data),
		Source:     entry.Source,
	}, nil
}

func (s Store) ReadIndex(provider, sessionID string) (Index, error) {
	dir, err := s.sessionDir(provider, sessionID)
	if err != nil {
		return Index{}, err
	}
	data, err := os.ReadFile(filepath.Join(dir, IndexFile))
	if err != nil {
		return Index{}, fmt.Errorf("read evidence index: %w", err)
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return Index{}, fmt.Errorf("parse evidence index: %w", err)
	}
	return idx, nil
}

func (s Store) sessionDir(provider, sessionID string) (string, error) {
	root, err := filepath.Abs(s.Root)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, safeSegment(provider), safeSegment(sessionID))
	clean := filepath.Clean(dir)
	if !isWithin(root, clean) {
		return "", fmt.Errorf("session path escapes storage root")
	}
	if err := os.MkdirAll(clean, 0o700); err != nil {
		return "", fmt.Errorf("create evidence directory: %w", err)
	}
	return clean, nil
}

func isEvidenceWorthy(event session.SessionEvent) bool {
	if event.Tool != nil || event.EventType == "unknown" {
		return true
	}
	lower := strings.ToLower(event.Content + " " + event.EventType)
	for _, signal := range []string{"error", "warning", "rollback", "blocker", "blocked", "correction", "failed", "failure", "exit code", "git status", "tests passed", "tests failed", "final"} {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	return false
}

func summarizeEvent(event session.SessionEvent) string {
	if event.Tool != nil {
		name := event.Tool.Name
		if name == "" {
			name = event.EventType
		}
		body := firstNonEmpty(event.Tool.Result, event.Tool.Stdout, event.Tool.Stderr, event.Tool.Arguments, event.Content)
		status := event.Tool.Status
		if status == "" {
			status = "unknown"
		}
		return fmt.Sprintf("%s %s: %s", name, status, session.Truncate(strings.Join(strings.Fields(body), " "), 180))
	}
	return fmt.Sprintf("%s: %s", event.EventType, session.Truncate(strings.Join(strings.Fields(event.Content), " "), 180))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func resolveAllowedSource(path string, allowedRoots []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("evidence source path is empty")
	}
	candidate, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", fmt.Errorf("resolve evidence source: %w", err)
	}
	if len(allowedRoots) == 0 {
		return "", fmt.Errorf("allowed workspace/session roots are required for evidence expansion")
	}
	for _, root := range allowedRoots {
		if root == "" {
			continue
		}
		absRoot, err := filepath.Abs(filepath.Clean(root))
		if err != nil {
			continue
		}
		evalRoot, err := filepath.EvalSymlinks(absRoot)
		if err == nil {
			absRoot = evalRoot
		}
		if isWithin(absRoot, resolved) {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("evidence source is outside allowed roots: %s", path)
}

func readSourceRange(path string, start, end int64, limit int) ([]byte, error) {
	if start < 0 || end < start {
		return nil, fmt.Errorf("invalid source byte range")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open evidence source: %w", err)
	}
	defer f.Close()
	length := end - start
	if length == 0 {
		info, err := f.Stat()
		if err != nil {
			return nil, err
		}
		length = info.Size() - start
	}
	if length > int64(limit) {
		length = int64(limit)
	}
	data := make([]byte, length)
	n, err := f.ReadAt(data, start)
	if err != nil && n == 0 {
		return nil, fmt.Errorf("read evidence source: %w", err)
	}
	return data[:n], nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, append(data, '\n'))
}

func writeJSONL[T any](path string, values []T) error {
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	for _, value := range values {
		if err := enc.Encode(value); err != nil {
			return err
		}
	}
	return atomicWrite(path, []byte(b.String()))
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(tmpName)
		if writeErr != nil {
			return fmt.Errorf("write temp file: %w", writeErr)
		}
		return fmt.Errorf("close temp file: %w", closeErr)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func acquireLock(dir string) (func(), error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(dir, ".write.lock")
	deadline := time.Now().Add(5 * time.Second)
	for {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !os.IsExist(err) || time.Now().After(deadline) {
			return nil, fmt.Errorf("acquire evidence lock: %w", err)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func safeSegment(value string) string {
	if value == "" {
		return "_"
	}
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func isWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)))
}
