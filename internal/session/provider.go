package session

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	ProviderClaudeCode  = "claude_code"
	ProviderCodex       = "codex"
	ProviderAntigravity = "antigravity"
)

// SessionProvider is the normalized provider adapter surface used by the
// cross-agent session broker. Provider packages own their on-disk format; core
// packages consume only these normalized types.
type SessionProvider interface {
	Name() string
	Discover() ([]SessionRef, error)
	Inspect(SessionRef) (SessionMetadata, error)
	Parse(SessionRef) ([]SessionEvent, []ParseError, error)
}

type SessionRef struct {
	ID          string
	Provider    string
	Path        string
	ProjectPath string
	StartTime   string
	ModTime     time.Time
	FirstPrompt string
}

type SessionMetadata struct {
	Ref                   SessionRef
	CWD                   string
	Originator            string
	CLI                   string
	ModelProvider         string
	UserMessageCount      int
	AssistantMessageCount int
	ToolCallCount         int
	ToolResultCount       int
	LineCount             int
	ParseErrors           []ParseError
}

type SessionEvent struct {
	EventID   string
	SessionID string
	Provider  string
	Timestamp string
	Sequence  int
	Role      string
	EventType string
	Content   string
	Tool      *SessionTool
	Source    EventSource
	Metadata  map[string]any
}

type SessionTool struct {
	CallID      string
	Name        string
	Namespace   string
	Arguments   string
	Result      string
	Stdout      string
	Stderr      string
	ExitCode    int
	HasExitCode bool
	Status      string
	DurationMS  int
}

type EventSource struct {
	Path        string
	LineStart   int
	LineEnd     int
	ByteStart   int64
	ByteEnd     int64
	ContentHash string
}

type ParseError struct {
	Path      string
	Line      int
	ByteStart int64
	Message   string
}

// StableEventID derives a reproducible event identifier from source position
// and event identity. It deliberately excludes filtered/redacted content so IDs
// remain stable across summarizer or redaction changes.
func StableEventID(sessionID, provider string, sequence int, source EventSource, eventType string) string {
	input := fmt.Sprintf("%s|%s|%d|%s|%d|%d|%s",
		sessionID, provider, sequence, source.Path, source.LineStart, source.ByteStart, eventType)
	sum := sha256.Sum256([]byte(input))
	return "ev-" + hex.EncodeToString(sum[:])[:16]
}

func ContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func IsSupportedProvider(provider string) bool {
	switch strings.ToLower(provider) {
	case ProviderClaudeCode, ProviderCodex, ProviderAntigravity, "all", "auto":
		return true
	default:
		return false
	}
}
