package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/analyzer"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/codexcodec"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func cmdInspect(args []string, reader session.TranscriptReader) {
	exitOnError(runInspect(args, os.Stdout, os.Stderr, parser.DefaultStore(), reader))
}

func runInspect(args []string, out io.Writer, errOut io.Writer, store parser.Store, reader session.TranscriptReader) error {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(errOut)
	provider := fs.String("provider", providerAuto, "session provider: auto, claude_code, codex, antigravity")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("session_id is required")
	}
	switch normalizeProvider(*provider) {
	case providerAuto, providerCodex:
		codec := codexcodec.Codec{}
		ref, err := codec.Resolve(fs.Arg(0))
		if err == nil {
			return renderCodexInspect(out, codec, ref)
		}
		if normalizeProvider(*provider) == providerCodex {
			return err
		}
	case providerAntigravity:
		return fmt.Errorf("antigravity provider is recognized but session parsing is not implemented yet")
	case providerClaudeCode:
		return renderClaudeInspect(out, store, reader, fs.Arg(0))
	default:
		return fmt.Errorf("unknown provider %q", *provider)
	}
	return renderClaudeInspect(out, store, reader, fs.Arg(0))
}

func renderCodexInspect(out io.Writer, codec codexcodec.Codec, ref session.SessionRef) error {
	meta, err := codec.Inspect(ref)
	if err != nil {
		return err
	}
	events, err := codec.ReadAll(ref.Path)
	if err != nil {
		return err
	}
	stats := analyzer.ComputeStats(events)
	fmt.Fprintf(out, "Provider: %s\n", ref.Provider)
	fmt.Fprintf(out, "Session: %s\n", ref.ID)
	fmt.Fprintf(out, "Path: %s\n", ref.Path)
	fmt.Fprintf(out, "CWD: %s\n", meta.CWD)
	fmt.Fprintf(out, "Started: %s\n", ref.StartTime)
	fmt.Fprintf(out, "Messages: user=%d assistant=%d\n", meta.UserMessageCount, meta.AssistantMessageCount)
	fmt.Fprintf(out, "Tools: calls=%d results=%d\n", meta.ToolCallCount, meta.ToolResultCount)
	fmt.Fprintf(out, "Raw chars: %s\n", analyzer.FormatNumber(stats.RawChars))
	fmt.Fprintf(out, "Filtered chars: %s\n", analyzer.FormatNumber(stats.FilteredChars))
	if stats.RawChars > 0 {
		saved := stats.RawChars - stats.FilteredChars
		fmt.Fprintf(out, "Saved: %s (%.1f%%)\n", analyzer.FormatNumber(saved), float64(saved)*100/float64(stats.RawChars))
	}
	if len(meta.ParseErrors) > 0 {
		fmt.Fprintf(out, "Parse errors: %d\n", len(meta.ParseErrors))
	}
	return nil
}

func renderClaudeInspect(out io.Writer, store parser.Store, reader session.TranscriptReader, prefix string) error {
	resolved, err := store.ResolveSession(prefix)
	if err != nil {
		return err
	}
	events, err := reader.ReadAll(resolved.Path)
	if err != nil {
		return err
	}
	stats := analyzer.ComputeStats(events)
	info, _ := os.Stat(resolved.Path)
	fmt.Fprintln(out, "Provider: claude_code")
	fmt.Fprintf(out, "Session: %s\n", resolved.ID)
	fmt.Fprintf(out, "Path: %s\n", resolved.Path)
	if info != nil {
		fmt.Fprintf(out, "Transcript: %.1fKB\n", float64(info.Size())/1024.0)
	}
	fmt.Fprintf(out, "Project: %s\n", filepath.Base(filepath.Dir(resolved.Path)))
	fmt.Fprintf(out, "Raw chars: %s\n", analyzer.FormatNumber(stats.RawChars))
	fmt.Fprintf(out, "Filtered chars: %s\n", analyzer.FormatNumber(stats.FilteredChars))
	return nil
}
