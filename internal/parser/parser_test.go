package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFormatTimestamp(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "ISO 8601 with Z suffix", input: "2025-03-15T14:30:00Z", want: "03-15 14:30"},
		{name: "ISO 8601 with positive offset", input: "2025-03-15T14:30:00+08:00", want: "03-15 14:30"},
		{name: "ISO 8601 with negative offset", input: "2025-12-01T09:05:00-05:00", want: "12-01 09:05"},
		{name: "ISO 8601 with milliseconds", input: "2025-06-20T23:59:59.123+00:00", want: "06-20 23:59"},
		{name: "ISO 8601 with microseconds", input: "2025-01-01T00:00:00.000000+00:00", want: "01-01 00:00"},
		{name: "invalid string returns placeholder", input: "not-a-timestamp", want: "??-?? ??:??"},
		{name: "empty string returns placeholder", input: "", want: "??-?? ??:??"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTimestamp(tt.input)
			if got != tt.want {
				t.Fatalf("FormatTimestamp(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStoreFindTranscript(t *testing.T) {
	root := t.TempDir()
	projectsDir := filepath.Join(root, "projects", "proj")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("create projects dir: %v", err)
	}
	sid := "12345678-1234-1234-1234-123456789abc"
	wantPath := filepath.Join(projectsDir, sid+".jsonl")
	writeFile(t, wantPath, "")

	got, err := (Store{ProjectsDir: filepath.Join(root, "projects")}).FindTranscript(sid)
	if err != nil {
		t.Fatalf("FindTranscript returned error: %v", err)
	}
	if got != wantPath {
		t.Fatalf("FindTranscript = %q, want %q", got, wantPath)
	}
}

func TestStoreFindTranscript_WhenProjectsDirIsMissing_ThenReturnsError(t *testing.T) {
	_, err := (Store{ProjectsDir: filepath.Join(t.TempDir(), "missing")}).FindTranscript("abc")
	if err == nil {
		t.Fatal("FindTranscript returned nil error, want walk error")
	}
	if !strings.Contains(err.Error(), "walk projects dir") {
		t.Fatalf("error = %v, want walk projects dir", err)
	}
}

func TestStoreResolveSession_WhenPrefixMatchesOne_ThenReturnsBothIDAndPath(t *testing.T) {
	root := t.TempDir()
	projectsDir := filepath.Join(root, "projects", "proj")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("create projects dir: %v", err)
	}
	sid := "12345678-1234-1234-1234-123456789abc"
	wantPath := filepath.Join(projectsDir, sid+".jsonl")
	writeFile(t, wantPath, "")

	store := Store{ProjectsDir: filepath.Join(root, "projects")}
	got, err := store.ResolveSession("12345678")
	if err != nil {
		t.Fatalf("ResolveSession returned error: %v", err)
	}
	if got.ID != sid {
		t.Fatalf("ResolveSession().ID = %q, want %q", got.ID, sid)
	}
	if got.Path != wantPath {
		t.Fatalf("ResolveSession().Path = %q, want %q", got.Path, wantPath)
	}
}

func TestStoreResolveSession_WhenFullUUID_ThenReturnsBothIDAndPath(t *testing.T) {
	root := t.TempDir()
	projectsDir := filepath.Join(root, "projects", "proj")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("create projects dir: %v", err)
	}
	sid := "12345678-1234-1234-1234-123456789abc"
	wantPath := filepath.Join(projectsDir, sid+".jsonl")
	writeFile(t, wantPath, "")

	store := Store{ProjectsDir: filepath.Join(root, "projects")}
	got, err := store.ResolveSession(sid)
	if err != nil {
		t.Fatalf("ResolveSession returned error: %v", err)
	}
	if got.ID != sid {
		t.Fatalf("ResolveSession().ID = %q, want %q", got.ID, sid)
	}
	if got.Path != wantPath {
		t.Fatalf("ResolveSession().Path = %q, want %q", got.Path, wantPath)
	}
}

func TestStoreResolveSession_WhenPrefixIsAmbiguous_ThenReturnsError(t *testing.T) {
	root := t.TempDir()
	projectsDir := filepath.Join(root, "projects", "proj")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("create projects dir: %v", err)
	}
	writeFile(t, filepath.Join(projectsDir, "12345678-0000-0000-0000-000000000000.jsonl"), "")
	writeFile(t, filepath.Join(projectsDir, "12345678-1111-1111-1111-111111111111.jsonl"), "")

	store := Store{ProjectsDir: filepath.Join(root, "projects")}
	_, err := store.ResolveSession("12345678")
	if err == nil {
		t.Fatal("ResolveSession returned nil error, want ambiguous error")
	}
	if !strings.Contains(err.Error(), "ambiguous prefix") {
		t.Fatalf("error = %v, want ambiguous prefix", err)
	}
}

func TestStoreResolveSession_WhenPrefixHasNoMatch_ThenReturnsError(t *testing.T) {
	root := t.TempDir()
	projectsDir := filepath.Join(root, "projects")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatalf("create projects dir: %v", err)
	}

	store := Store{ProjectsDir: projectsDir}
	_, err := store.ResolveSession("notfound")
	if err == nil {
		t.Fatal("ResolveSession returned nil error, want not found error")
	}
	if !strings.Contains(err.Error(), "session prefix not found") {
		t.Fatalf("error = %v, want session prefix not found", err)
	}
}

func TestStoreLoadSessionMeta(t *testing.T) {
	root := t.TempDir()
	metaDir := filepath.Join(root, "session-meta")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatalf("create meta dir: %v", err)
	}
	sid := "12345678-1234-1234-1234-123456789abc"
	writeFile(t, filepath.Join(metaDir, sid+".json"), `{"session_id":"`+sid+`","duration_minutes":3}`)

	meta, err := (Store{SessionMetaDir: metaDir}).LoadSessionMeta(sid)
	if err != nil {
		t.Fatalf("LoadSessionMeta returned error: %v", err)
	}
	if meta["session_id"] != sid {
		t.Fatalf("session_id = %#v, want %q", meta["session_id"], sid)
	}
}

func TestStoreLoadSessionMeta_WhenJSONIsInvalid_ThenReturnsError(t *testing.T) {
	root := t.TempDir()
	metaDir := filepath.Join(root, "session-meta")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatalf("create meta dir: %v", err)
	}
	sid := "12345678-1234-1234-1234-123456789abc"
	writeFile(t, filepath.Join(metaDir, sid+".json"), `{"session_id":`)

	_, err := (Store{SessionMetaDir: metaDir}).LoadSessionMeta(sid)
	if err == nil {
		t.Fatal("LoadSessionMeta returned nil error, want parse error")
	}
	if !strings.Contains(err.Error(), "parse session meta") {
		t.Fatalf("error = %v, want parse session meta", err)
	}
}

func TestStoreListSessionMetaFiles_SortsNewestFirst(t *testing.T) {
	root := t.TempDir()
	metaDir := filepath.Join(root, "session-meta")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatalf("create meta dir: %v", err)
	}
	oldPath := filepath.Join(metaDir, "old.json")
	newPath := filepath.Join(metaDir, "new.json")
	writeFile(t, oldPath, `{}`)
	writeFile(t, newPath, `{}`)
	oldTime := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	newTime := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(oldPath, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes old: %v", err)
	}
	if err := os.Chtimes(newPath, newTime, newTime); err != nil {
		t.Fatalf("chtimes new: %v", err)
	}

	files, err := (Store{SessionMetaDir: metaDir}).ListSessionMetaFiles()
	if err != nil {
		t.Fatalf("ListSessionMetaFiles returned error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("file count = %d, want 2", len(files))
	}
	if filepath.Base(files[0].Path) != "new.json" || filepath.Base(files[1].Path) != "old.json" {
		t.Fatalf("files order = %#v, want newest first", files)
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
