// Package codexcodec parses Codex rollout JSONL sessions into both the new
// normalized provider schema and the existing analyzer event model.
package codexcodec

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

type Codec struct {
	Roots []string
}

func (c Codec) Name() string { return session.ProviderCodex }

func (c Codec) Discover() ([]session.SessionRef, error) {
	roots := c.sessionRoots()
	var refs []session.SessionRef
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
				return nil
			}
			ref, err := c.inspectRef(path)
			if err == nil {
				refs = append(refs, ref)
			}
			return nil
		})
	}
	sort.Slice(refs, func(i, j int) bool {
		ti, tj := parseTime(refs[i].StartTime), parseTime(refs[j].StartTime)
		if !ti.IsZero() && !tj.IsZero() {
			return ti.After(tj)
		}
		return refs[i].ModTime.After(refs[j].ModTime)
	})
	if len(refs) == 0 {
		return nil, fmt.Errorf("no Codex sessions found; configure CODEX_SESSION_ROOTS or session_sources.codex.roots")
	}
	return refs, nil
}

func (c Codec) Inspect(ref session.SessionRef) (session.SessionMetadata, error) {
	if ref.Path == "" {
		resolved, err := c.Resolve(ref.ID)
		if err != nil {
			return session.SessionMetadata{}, err
		}
		ref = resolved
	}
	events, parseErrors, err := c.Parse(ref)
	if err != nil {
		return session.SessionMetadata{}, err
	}
	meta := session.SessionMetadata{Ref: ref, ParseErrors: parseErrors}
	for _, event := range events {
		if event.Sequence > meta.LineCount {
			meta.LineCount = event.Sequence
		}
		switch event.EventType {
		case "message":
			switch event.Role {
			case "user":
				meta.UserMessageCount++
			case "assistant":
				meta.AssistantMessageCount++
			}
		case "tool_call":
			meta.ToolCallCount++
		case "tool_result":
			meta.ToolResultCount++
		case "session_meta":
			if v, ok := event.Metadata["cwd"].(string); ok {
				meta.CWD = v
			}
			if v, ok := event.Metadata["originator"].(string); ok {
				meta.Originator = v
			}
			if v, ok := event.Metadata["cli_version"].(string); ok {
				meta.CLI = v
			}
			if v, ok := event.Metadata["model_provider"].(string); ok {
				meta.ModelProvider = v
			}
		}
	}
	return meta, nil
}

func (c Codec) Parse(ref session.SessionRef) ([]session.SessionEvent, []session.ParseError, error) {
	if ref.Path == "" {
		resolved, err := c.Resolve(ref.ID)
		if err != nil {
			return nil, nil, err
		}
		ref = resolved
	}
	f, err := os.Open(ref.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("open codex session: %w", err)
	}
	defer f.Close()

	var events []session.SessionEvent
	var parseErrors []session.ParseError
	reader := bufio.NewReader(f)
	var byteStart int64
	lineNo := 0
	for {
		line, readErr := reader.ReadBytes('\n')
		if len(line) == 0 && readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.EOF {
			return nil, parseErrors, fmt.Errorf("read codex session: %w", readErr)
		}
		lineNo++
		trimmed := bytes.TrimSpace(line)
		byteEnd := byteStart + int64(len(line))
		if len(trimmed) > 0 {
			event, ok, parseErr := c.parseLine(ref, lineNo, byteStart, byteEnd, line)
			if parseErr != nil {
				parseErrors = append(parseErrors, session.ParseError{
					Path:      ref.Path,
					Line:      lineNo,
					ByteStart: byteStart,
					Message:   parseErr.Error(),
				})
			} else if ok {
				event.Sequence = lineNo
				event.EventID = session.StableEventID(ref.ID, c.Name(), event.Sequence, event.Source, event.EventType)
				events = append(events, event)
			}
		}
		byteStart = byteEnd
		if readErr == io.EOF {
			break
		}
	}
	return events, parseErrors, nil
}

func (c Codec) ReadAll(path string) ([]session.Event, error) {
	ref, err := c.inspectRef(path)
	if err != nil {
		ref = session.SessionRef{ID: strings.TrimSuffix(filepath.Base(path), ".jsonl"), Provider: c.Name(), Path: path}
	}
	events, _, err := c.Parse(ref)
	if err != nil {
		return nil, err
	}
	return legacyEvents(events), nil
}

func (c Codec) Resolve(prefix string) (session.SessionRef, error) {
	if prefix == "" {
		return session.SessionRef{}, fmt.Errorf("session_id is required")
	}
	refs, err := c.Discover()
	if err != nil {
		return session.SessionRef{}, err
	}
	var matches []session.SessionRef
	for _, ref := range refs {
		if strings.HasPrefix(ref.ID, prefix) || strings.HasPrefix(filepath.Base(ref.Path), prefix) {
			matches = append(matches, ref)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		shown := matches
		if len(shown) > 5 {
			shown = shown[:5]
		}
		ids := make([]string, len(shown))
		for i, ref := range shown {
			ids[i] = session.ShortID(ref.ID, 12)
		}
		return session.SessionRef{}, fmt.Errorf("ambiguous Codex session prefix %q, matches: %s", prefix, strings.Join(ids, ", "))
	}
	return session.SessionRef{}, fmt.Errorf("Codex session prefix not found: %s", prefix)
}

func (c Codec) sessionRoots() []string {
	if len(c.Roots) > 0 {
		return c.Roots
	}
	if env := os.Getenv("CODEX_SESSION_ROOTS"); env != "" {
		return splitRoots(env)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{filepath.Join(home, ".codex", "sessions")}
}

func splitRoots(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == rune(os.PathListSeparator)
	})
	var roots []string
	for _, field := range fields {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			roots = append(roots, trimmed)
		}
	}
	return roots
}

func (c Codec) inspectRef(path string) (session.SessionRef, error) {
	info, err := os.Stat(path)
	if err != nil {
		return session.SessionRef{}, err
	}
	ref := session.SessionRef{
		ID:       strings.TrimSuffix(filepath.Base(path), ".jsonl"),
		Provider: c.Name(),
		Path:     path,
		ModTime:  info.ModTime(),
	}

	f, err := os.Open(path)
	if err != nil {
		return ref, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	lines := 0
	for scanner.Scan() && lines < 80 {
		lines++
		var env envelope
		if err := json.Unmarshal(scanner.Bytes(), &env); err != nil {
			continue
		}
		if env.Type == "session_meta" {
			var meta sessionMetaPayload
			if json.Unmarshal(env.Payload, &meta) == nil {
				if meta.ID != "" {
					ref.ID = meta.ID
				} else if meta.SessionID != "" {
					ref.ID = meta.SessionID
				}
				ref.ProjectPath = meta.CWD
				ref.StartTime = firstNonEmpty(env.Timestamp, meta.Timestamp)
			}
			continue
		}
		var payload typedPayload
		if json.Unmarshal(env.Payload, &payload) == nil && payload.Type == "user_message" && ref.FirstPrompt == "" {
			ref.FirstPrompt = truncateLine(payload.Message, 80)
		}
	}
	return ref, nil
}

func (c Codec) parseLine(ref session.SessionRef, lineNo int, byteStart, byteEnd int64, line []byte) (session.SessionEvent, bool, error) {
	var env envelope
	if err := json.Unmarshal(line, &env); err != nil {
		return session.SessionEvent{}, false, fmt.Errorf("parse codex jsonl line: %w", err)
	}
	source := session.EventSource{
		Path:        ref.Path,
		LineStart:   lineNo,
		LineEnd:     lineNo,
		ByteStart:   byteStart,
		ByteEnd:     byteEnd,
		ContentHash: session.ContentHash(bytes.TrimRight(line, "\r\n")),
	}
	base := session.SessionEvent{
		SessionID: ref.ID,
		Provider:  c.Name(),
		Timestamp: env.Timestamp,
		Source:    source,
		Metadata:  map[string]any{},
	}

	switch env.Type {
	case "session_meta":
		var meta map[string]any
		_ = json.Unmarshal(env.Payload, &meta)
		base.EventType = "session_meta"
		base.Metadata = meta
		return base, true, nil
	case "turn_context":
		base.EventType = "turn_context"
		base.Content = compactJSON(env.Payload)
		return base, true, nil
	case "event_msg":
		return parseEventMsg(base, env.Payload), true, nil
	case "response_item":
		return parseResponseItem(base, env.Payload), true, nil
	default:
		base.EventType = "unknown"
		base.Content = compactJSON(env.Payload)
		base.Metadata["outer_type"] = env.Type
		return base, true, nil
	}
}
