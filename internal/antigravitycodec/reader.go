// Package antigravitycodec parses Google Antigravity standalone-app
// conversation transcripts from ~/.gemini/antigravity/brain.
package antigravitycodec

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

func (c Codec) Name() string { return session.ProviderAntigravity }

func (c Codec) Discover() ([]session.SessionRef, error) {
	var refs []session.SessionRef
	seenLogs := map[string]bool{}
	for _, root := range c.sessionRoots() {
		info, err := os.Stat(root)
		if err != nil || !info.IsDir() {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := filepath.Base(path)
			if name != "transcript_full.jsonl" && name != "transcript.jsonl" {
				return nil
			}
			logDir := filepath.Dir(path)
			if name == "transcript.jsonl" {
				full := filepath.Join(logDir, "transcript_full.jsonl")
				if _, err := os.Stat(full); err == nil {
					return nil
				}
			}
			if seenLogs[logDir] {
				return nil
			}
			seenLogs[logDir] = true
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
		return nil, fmt.Errorf("no Antigravity standalone app sessions found; configure ANTIGRAVITY_SESSION_ROOTS or install/use Google Antigravity")
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
	meta := session.SessionMetadata{Ref: ref, CWD: ref.ProjectPath, ParseErrors: parseErrors}
	for _, event := range events {
		if event.Source.LineStart > meta.LineCount {
			meta.LineCount = event.Source.LineStart
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
		}
	}
	meta.CLI = "Google Antigravity"
	meta.ModelProvider = "google"
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
		return nil, nil, fmt.Errorf("open Antigravity transcript: %w", err)
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
			return nil, parseErrors, fmt.Errorf("read Antigravity transcript: %w", readErr)
		}
		lineNo++
		trimmed := bytes.TrimSpace(line)
		byteEnd := byteStart + int64(len(line))
		if len(trimmed) > 0 {
			parsed, parseErr := c.parseLine(ref, lineNo, byteStart, byteEnd, line)
			if parseErr != nil {
				parseErrors = append(parseErrors, session.ParseError{
					Path:      ref.Path,
					Line:      lineNo,
					ByteStart: byteStart,
					Message:   parseErr.Error(),
				})
			} else {
				for _, event := range parsed {
					event.Sequence = len(events) + 1
					event.EventID = session.StableEventID(ref.ID, c.Name(), event.Sequence, event.Source, event.EventType)
					events = append(events, event)
				}
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
		ref = session.SessionRef{ID: sessionIDFromPath(path), Provider: c.Name(), Path: path}
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
		if strings.HasPrefix(ref.ID, prefix) {
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
		return session.SessionRef{}, fmt.Errorf("ambiguous Antigravity session prefix %q, matches: %s", prefix, strings.Join(ids, ", "))
	}
	return session.SessionRef{}, fmt.Errorf("Antigravity session prefix not found: %s", prefix)
}

func (c Codec) sessionRoots() []string {
	if len(c.Roots) > 0 {
		return c.Roots
	}
	if env := os.Getenv("ANTIGRAVITY_SESSION_ROOTS"); env != "" {
		return splitRoots(env)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return nil
	}
	return []string{filepath.Join(home, ".gemini", "antigravity", "brain")}
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
		ID:       sessionIDFromPath(path),
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
	for scanner.Scan() && lines < 200 {
		lines++
		var step transcriptStep
		if err := json.Unmarshal(scanner.Bytes(), &step); err != nil {
			continue
		}
		if ref.StartTime == "" && step.CreatedAt != "" {
			ref.StartTime = step.CreatedAt
		}
		if ref.FirstPrompt == "" && step.Type == "USER_INPUT" && step.Source == "USER_EXPLICIT" {
			ref.FirstPrompt = truncateLine(firstNonEmpty(extractTagged(step.Content, "USER_REQUEST"), step.Content), 80)
		}
		if ref.ProjectPath == "" {
			for _, call := range step.ToolCalls {
				if cwd := stringArg(call.Args, "Cwd", "cwd"); cwd != "" {
					ref.ProjectPath = cwd
					break
				}
			}
		}
	}
	return ref, nil
}

func (c Codec) parseLine(ref session.SessionRef, lineNo int, byteStart, byteEnd int64, line []byte) ([]session.SessionEvent, error) {
	var step transcriptStep
	if err := json.Unmarshal(line, &step); err != nil {
		return nil, fmt.Errorf("parse Antigravity jsonl line: %w", err)
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
		Source:    source,
		Metadata:  map[string]any{},
	}
	return parseStep(base, step), nil
}

func sessionIDFromPath(path string) string {
	dir := filepath.Dir(path)
	for {
		if filepath.Base(dir) == ".system_generated" {
			return filepath.Base(filepath.Dir(dir))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
}
