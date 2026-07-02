package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/evidence"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func cmdEvidence(args []string, reader session.TranscriptReader) {
	exitOnError(runEvidence(args, os.Stdout, os.Stderr, parser.DefaultStore(), reader))
}

func runEvidence(args []string, out io.Writer, errOut io.Writer, _ parser.Store, _ session.TranscriptReader) error {
	fs := flag.NewFlagSet("evidence", flag.ContinueOnError)
	fs.SetOutput(errOut)
	provider := fs.String("provider", providerAuto, "provider that owns the evidence artifact")
	configPath := fs.String("config", "", "path to session-context config.json")
	limit := fs.Int("limit", 64*1024, "maximum source bytes to expand")
	unredacted := fs.Bool("unredacted", false, "return unredacted source bytes; disabled by default")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: cc-session evidence <session-id> <evidence-id>")
	}
	cfg := config.LoadSessionContext()
	if *configPath != "" {
		cfg = config.LoadSessionContextFromPath(*configPath)
	}
	result, err := evidence.Store{Root: cfg.StorageRoot}.Expand(evidence.ExpandOptions{
		Provider:     normalizeProvider(*provider),
		SessionID:    fs.Arg(0),
		EvidenceID:   fs.Arg(1),
		AllowedRoots: allowedEvidenceRoots(cfg),
		Limit:        *limit,
		Unredacted:   *unredacted,
	})
	if err != nil {
		return err
	}
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	return enc.Encode(result)
}

func allowedEvidenceRoots(cfg config.SessionContextConfig) []string {
	roots := append([]string{}, cfg.AllowedWorkspaceRoot...)
	for _, source := range cfg.SessionSources {
		roots = append(roots, source.Roots...)
	}
	return roots
}
