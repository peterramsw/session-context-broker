package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/broker"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func cmdVerifyWorkspace(args []string, reader session.TranscriptReader) {
	exitOnError(runVerifyWorkspace(args, os.Stdout, os.Stderr, parser.DefaultStore(), reader))
}

func runVerifyWorkspace(args []string, out io.Writer, errOut io.Writer, store parser.Store, reader session.TranscriptReader) error {
	fs := flag.NewFlagSet("verify-workspace", flag.ContinueOnError)
	fs.SetOutput(errOut)
	configPath := fs.String("config", "", "path to session-context config.json")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: cc-session verify-workspace <path>")
	}
	cfg := config.LoadSessionContext()
	if *configPath != "" {
		cfg = config.LoadSessionContextFromPath(*configPath)
	}
	report, err := broker.New(store, reader, cfg).VerifyWorkspace(fs.Arg(0))
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	return enc.Encode(report)
}
