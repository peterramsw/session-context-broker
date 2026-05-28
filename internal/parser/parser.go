// Package parser handles session discovery and metadata I/O.
package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Store points at Claude Code's on-disk session data.
type Store struct {
	ProjectsDir    string
	SessionMetaDir string
}

// DefaultStore returns a Store derived from the current user's ~/.claude.
func DefaultStore() Store {
	claudeDir := filepath.Join(homeDir(), ".claude")
	return Store{
		ProjectsDir:    filepath.Join(claudeDir, "projects"),
		SessionMetaDir: filepath.Join(claudeDir, "usage-data", "session-meta"),
	}
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// FindTranscript locates a transcript JSONL file by session ID under the store's projects dir.
func (s Store) FindTranscript(sessionID string) (string, error) {
	var found string
	err := filepath.Walk(s.ProjectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if base == sessionID+".jsonl" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk projects dir: %w", err)
	}
	return found, nil
}

// ResolvedSession holds the session ID and transcript path resolved in a single walk.
type ResolvedSession struct {
	ID   string
	Path string
}

// ResolveSession resolves a prefix to a full session UUID and its transcript path
// in a single filesystem walk, avoiding the double-walk of ResolveSessionID + FindTranscript.
func (s Store) ResolveSession(prefix string) (ResolvedSession, error) {
	if len(prefix) == 36 {
		path, err := s.FindTranscript(prefix)
		if err != nil {
			return ResolvedSession{}, err
		}
		return ResolvedSession{ID: prefix, Path: path}, nil
	}

	type match struct {
		id   string
		path string
	}
	var matches []match
	err := filepath.Walk(s.ProjectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".jsonl" {
			stem := strings.TrimSuffix(filepath.Base(path), ".jsonl")
			if strings.HasPrefix(stem, prefix) {
				matches = append(matches, match{id: stem, path: path})
			}
		}
		return nil
	})
	if err != nil {
		return ResolvedSession{}, fmt.Errorf("walk projects dir: %w", err)
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].id < matches[j].id })

	if len(matches) == 1 {
		return ResolvedSession{ID: matches[0].id, Path: matches[0].path}, nil
	}
	if len(matches) > 1 {
		shown := matches
		if len(shown) > 5 {
			shown = shown[:5]
		}
		shortIDs := make([]string, len(shown))
		for i, m := range shown {
			if len(m.id) >= 12 {
				shortIDs[i] = m.id[:12]
			} else {
				shortIDs[i] = m.id
			}
		}
		return ResolvedSession{}, fmt.Errorf("ambiguous prefix '%s', matches: %s", prefix, strings.Join(shortIDs, ", "))
	}
	return ResolvedSession{}, fmt.Errorf("session prefix not found: %s", prefix)
}

// LoadSessionMeta reads session metadata from the store's session-meta directory.
func (s Store) LoadSessionMeta(sessionID string) (map[string]interface{}, error) {
	metaFile := filepath.Join(s.SessionMetaDir, sessionID+".json")
	data, err := os.ReadFile(metaFile)
	if err != nil {
		return nil, err
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parse session meta %s: %w", sessionID, err)
	}
	return meta, nil
}

// ResolveSessionID resolves a prefix to a full session UUID in the store.
func (s Store) ResolveSessionID(prefix string) (string, error) {
	if len(prefix) == 36 {
		return prefix, nil
	}

	var matches []string
	err := filepath.Walk(s.ProjectsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".jsonl" {
			stem := strings.TrimSuffix(filepath.Base(path), ".jsonl")
			if strings.HasPrefix(stem, prefix) {
				matches = append(matches, stem)
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("walk projects dir: %w", err)
	}
	sort.Strings(matches)

	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		shown := matches
		if len(shown) > 5 {
			shown = shown[:5]
		}
		shortIDs := make([]string, len(shown))
		for i, m := range shown {
			if len(m) >= 12 {
				shortIDs[i] = m[:12]
			} else {
				shortIDs[i] = m
			}
		}
		return "", fmt.Errorf("ambiguous prefix '%s', matches: %s", prefix, strings.Join(shortIDs, ", "))
	}
	return "", fmt.Errorf("session prefix not found: %s", prefix)
}

// SessionMetaFile holds metadata about a session, used for listing.
type SessionMetaFile struct {
	Path    string
	ModTime time.Time
}

// ListSessionMetaFiles returns session meta files sorted by modification time (newest first).
func (s Store) ListSessionMetaFiles() ([]SessionMetaFile, error) {
	entries, err := os.ReadDir(s.SessionMetaDir)
	if err != nil {
		return nil, fmt.Errorf("read session meta dir: %w", err)
	}

	var files []SessionMetaFile
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, SessionMetaFile{
			Path:    filepath.Join(s.SessionMetaDir, e.Name()),
			ModTime: info.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})
	return files, nil
}

func parseISO(s string) (time.Time, error) {
	formats := []string{
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05.000-07:00",
		"2006-01-02T15:04:05.000000-07:00",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable timestamp: %s", s)
}
