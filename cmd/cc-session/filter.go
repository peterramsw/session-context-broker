package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/analyzer"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/codexcodec"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func cmdFilter(args []string, reader session.TranscriptReader) {
	exitOnError(runFilter(args, os.Stdout, os.Stderr, parser.DefaultStore(), reader))
}

func runFilter(args []string, out io.Writer, errOut io.Writer, store parser.Store, reader session.TranscriptReader) error {
	fs := flag.NewFlagSet("filter", flag.ContinueOnError)
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
			return renderFiltered(out, ref.Path, codec)
		}
		if normalizeProvider(*provider) == providerCodex {
			return err
		}
	case providerAntigravity:
		return fmt.Errorf("antigravity provider is recognized but session parsing is not implemented yet")
	case providerClaudeCode:
		resolved, err := store.ResolveSession(fs.Arg(0))
		if err != nil {
			return err
		}
		return renderFiltered(out, resolved.Path, reader)
	default:
		return fmt.Errorf("unknown provider %q", *provider)
	}
	resolved, err := store.ResolveSession(fs.Arg(0))
	if err != nil {
		return err
	}
	return renderFiltered(out, resolved.Path, reader)
}

func renderFiltered(out io.Writer, path string, reader session.TranscriptReader) error {
	events, err := reader.ReadAll(path)
	if err != nil {
		return err
	}
	result := analyzer.ComputeStats(events)
	_, err = fmt.Fprintln(out, result.FilteredText)
	return err
}
