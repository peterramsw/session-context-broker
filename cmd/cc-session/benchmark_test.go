package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/analyzer"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
)

func TestPrintCompressionSection_WhenRendered_ThenUsesNewSessionTotalContext(t *testing.T) {
	results := []sessionBenchResult{
		{
			shortID:          "aaaaaaaa",
			contextTokens:    100_000,
			filteredTokens:   23_456,
			newContextTokens: 60_000,
			savedPct:         40.0,
		},
		{
			shortID:          "bbbbbbbb",
			contextTokens:    200_000,
			filteredTokens:   87_654,
			newContextTokens: 120_000,
			savedPct:         40.0,
		},
	}

	var out bytes.Buffer
	printCompressionSection(&out, results)
	got := out.String()

	if !strings.Contains(got, "Context      NewCtx") {
		t.Fatalf("compression header must compare total contexts, got:\n%s", got)
	}
	if strings.Contains(got, "Filtered") {
		t.Fatalf("compression table must not label history-only tokens as the comparable total context:\n%s", got)
	}
	if !strings.Contains(got, "aaaaaaaa") ||
		!strings.Contains(got, "100,000") ||
		!strings.Contains(got, "60,000") ||
		!strings.Contains(got, "40.0%") {
		t.Fatalf("compression row missing new session total context:\n%s", got)
	}
	if strings.Contains(got, "23,456") || strings.Contains(got, "87,654") {
		t.Fatalf("compression row leaked filtered-history-only token count:\n%s", got)
	}
}

func TestMedianFloat64_GivenEvenCount_ThenAveragesMiddleValues(t *testing.T) {
	got := medianFloat64([]float64{78.9, 90.3})
	want := 84.6
	if got != want {
		t.Fatalf("medianFloat64(even) = %.1f, want %.1f", got, want)
	}
}

func TestPrintCompressionSection_GivenEvenCount_ThenPrintsAveragedMedian(t *testing.T) {
	results := []sessionBenchResult{
		{shortID: "aaaaaaaa", contextTokens: 100, newContextTokens: 20, savedPct: 78.9},
		{shortID: "bbbbbbbb", contextTokens: 100, newContextTokens: 10, savedPct: 90.3},
	}

	var out bytes.Buffer
	printCompressionSection(&out, results)

	if got := out.String(); !strings.Contains(got, "Median: 84.6%") {
		t.Fatalf("compression summary must average the two middle values for even counts:\n%s", got)
	}
}

func TestRunBenchmark_WhenSessionHasAPIUsage_ThenUsesTokenCountingAPIForNewContext(t *testing.T) {
	root := t.TempDir()
	sid := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	projectDir := filepath.Join(root, "projects", "proj")
	metaDir := filepath.Join(root, "session-meta")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project dir: %v", err)
	}
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatalf("create meta dir: %v", err)
	}

	transcript := strings.Join([]string{
		`{"type":"user","timestamp":"2026-05-28T00:00:00Z","message":{"role":"user","content":"hello"}}`,
		`{"type":"assistant","timestamp":"2026-05-28T00:00:01Z","message":{"role":"assistant","content":"hi","usage":{"input_tokens":100000,"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"output_tokens":1000}}}`,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(projectDir, sid+".jsonl"), []byte(transcript), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	writeListMeta(t, metaDir, sid, "/tmp/proj", "hello")

	const filteredTokenCount = 23_456
	original := countTokensFn
	t.Cleanup(func() { countTokensFn = original })
	var countedText string
	countTokensFn = func(text string) (int, error) {
		countedText = text
		return filteredTokenCount, nil
	}

	var stdout, stderr bytes.Buffer
	store := parser.Store{ProjectsDir: filepath.Join(root, "projects"), SessionMetaDir: metaDir}
	if err := runBenchmark([]string{"--n", "1", "--min-kb", "0", "--overhead", "40000"}, &stdout, &stderr, store, testReader); err != nil {
		t.Fatalf("runBenchmark returned error: %v", err)
	}

	if countedText == "" {
		t.Fatal("countTokensFn was not called")
	}
	wantNewContext := analyzer.FormatNumber(40_000 + filteredTokenCount)
	if got := stdout.String(); !strings.Contains(got, wantNewContext) {
		t.Fatalf("benchmark output missing NewCtx from token counting API (%s):\n%s", wantNewContext, got)
	}
}

func TestRunBenchmark_WhenSessionHasNoAPIUsage_ThenSkipsSession(t *testing.T) {
	root := t.TempDir()
	sid := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	projectDir := filepath.Join(root, "projects", "proj")
	metaDir := filepath.Join(root, "session-meta")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("create project dir: %v", err)
	}
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatalf("create meta dir: %v", err)
	}

	transcript := strings.Join([]string{
		`{"type":"user","timestamp":"2026-05-28T00:00:00Z","message":{"role":"user","content":"hello"}}`,
		`{"type":"assistant","timestamp":"2026-05-28T00:00:01Z","message":{"role":"assistant","content":"hi"}}`,
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(projectDir, sid+".jsonl"), []byte(transcript), 0o644); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	writeListMeta(t, metaDir, sid, "/tmp/proj", "hello")

	original := countTokensFn
	t.Cleanup(func() { countTokensFn = original })
	countTokensFn = func(text string) (int, error) {
		t.Fatal("countTokensFn must not be called for sessions without API usage data")
		return 0, nil
	}

	var stdout, stderr bytes.Buffer
	store := parser.Store{ProjectsDir: filepath.Join(root, "projects"), SessionMetaDir: metaDir}
	if err := runBenchmark([]string{"--n", "1", "--min-kb", "0"}, &stdout, &stderr, store, testReader); err != nil {
		t.Fatalf("runBenchmark returned error: %v", err)
	}

	got := stdout.String()
	if !strings.Contains(got, "missing API usage data") || !strings.Contains(got, "No sessions could be processed.") {
		t.Fatalf("benchmark output should skip sessions without API usage data:\n%s", got)
	}
}
