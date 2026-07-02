package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/broker"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func cmdSearch(args []string, reader session.TranscriptReader) {
	exitOnError(runSearch(args, os.Stdout, os.Stderr, parser.DefaultStore(), reader))
}

func runSearch(args []string, out io.Writer, errOut io.Writer, store parser.Store, reader session.TranscriptReader) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(errOut)
	provider := fs.String("provider", providerAuto, "session provider: auto, claude_code, codex, antigravity")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: cc-session search <session-id> <query>")
	}
	svc := broker.New(store, reader, config.LoadSessionContext())
	matches, err := svc.SearchSession(fs.Arg(0), normalizeProvider(*provider), fs.Arg(1))
	if err != nil {
		return err
	}
	for _, match := range matches {
		fmt.Fprintf(out, "%s  %-12s  %s\n", match.EvidenceID, match.EventType, match.Summary)
	}
	if len(matches) == 0 {
		fmt.Fprintln(out, "No matches found.")
	}
	return nil
}
