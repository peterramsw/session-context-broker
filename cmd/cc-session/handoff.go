package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/analyzer"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/codexcodec"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/distiller"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/handoff"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func cmdHandoff(args []string, reader session.TranscriptReader) {
	exitOnError(runHandoff(args, os.Stdout, os.Stderr, parser.DefaultStore(), reader))
}

func runHandoff(args []string, out io.Writer, errOut io.Writer, store parser.Store, reader session.TranscriptReader) error {
	fs := flag.NewFlagSet("handoff", flag.ContinueOnError)
	fs.SetOutput(errOut)
	provider := fs.String("provider", providerAuto, "session provider: auto, claude_code, codex")
	configPath := fs.String("config", "", "path to session-context config.json")
	storageRoot := fs.String("out", "", "override storage root for generated handoff artifacts")
	force := fs.Bool("force", false, "overwrite existing handoff artifacts")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("session_id is required")
	}

	cfg := config.LoadSessionContext()
	if *configPath != "" {
		cfg = config.LoadSessionContextFromPath(*configPath)
	}
	if *storageRoot != "" {
		cfg.StorageRoot = *storageRoot
	}
	if !cfg.LocalLLM.IsEnabled() {
		return fmt.Errorf("Local LLM is not enabled; configure local_llm.enabled=true with base_url/model, or provide legacy qwen config")
	}

	input, err := resolveHandoffSession(fs.Arg(0), normalizeProvider(*provider), store, reader)
	if err != nil {
		return err
	}
	logUsageAsync("handoff", session.ShortID(input.info.SessionID, 8))

	req := distiller.Request{
		Config:             cfg.LocalLLM,
		Session:            input.info,
		FilteredTranscript: input.filteredText,
	}
	generated, diag, err := distiller.Generate(context.Background(), req, distiller.NewClient(cfg.LocalLLM))
	if err != nil {
		var invalid distiller.InvalidOutputError
		if errors.As(err, &invalid) && cfg.StorageRoot != "" {
			if path, writeErr := handoff.WriteFailedRaw(cfg.StorageRoot, input.info.Provider, input.info.SessionID, invalid.Raw); writeErr == nil {
				fmt.Fprintf(errOut, "raw failed Local LLM output written to %s\n", path)
			}
		}
		return err
	}
	dir, err := handoff.WriteArtifacts(cfg.StorageRoot, generated, *force)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Provider: %s\n", input.info.Provider)
	fmt.Fprintf(out, "Session: %s\n", input.info.SessionID)
	fmt.Fprintf(out, "Model: %s\n", cfg.LocalLLM.Model)
	fmt.Fprintf(out, "Max context: %d\n", cfg.LocalLLM.MaxContext)
	fmt.Fprintf(out, "Max output tokens: %d\n", cfg.LocalLLM.MaxOutputTokens)
	fmt.Fprintf(out, "Temperature: %g\n", localLLMTemperature(cfg.LocalLLM))
	if cfg.LocalLLM.TopP != nil {
		fmt.Fprintf(out, "TopP: %g\n", *cfg.LocalLLM.TopP)
	}
	if cfg.LocalLLM.TopK > 0 {
		fmt.Fprintf(out, "TopK: %d\n", cfg.LocalLLM.TopK)
	}
	fmt.Fprintf(out, "Chunks: %d\n", diag.Chunks)
	fmt.Fprintf(out, "Repaired: %v\n", diag.Repaired)
	fmt.Fprintf(out, "Raw chars: %s\n", analyzer.FormatNumber(input.info.RawChars))
	fmt.Fprintf(out, "Filtered chars: %s\n", analyzer.FormatNumber(input.info.FilteredChars))
	fmt.Fprintf(out, "Redacted input chars: %s\n", analyzer.FormatNumber(diag.RedactedInputChars))
	fmt.Fprintf(out, "Output: %s\n", dir)
	return nil
}

func localLLMTemperature(cfg config.LocalLLMConfig) float64 {
	if cfg.Temperature == nil {
		return 0
	}
	return *cfg.Temperature
}

type handoffInput struct {
	info         handoff.SessionInfo
	filteredText string
}

func resolveHandoffSession(prefix string, provider string, store parser.Store, reader session.TranscriptReader) (handoffInput, error) {
	switch provider {
	case providerAuto, providerCodex:
		codec := codexcodec.Codec{}
		ref, err := codec.Resolve(prefix)
		if err == nil {
			events, err := codec.ReadAll(ref.Path)
			if err != nil {
				return handoffInput{}, err
			}
			meta, _ := codec.Inspect(ref)
			stats := analyzer.ComputeStats(events)
			workspace := meta.CWD
			if workspace == "" {
				workspace = ref.ProjectPath
			}
			return handoffInput{
				info: handoff.SessionInfo{
					Provider:      session.ProviderCodex,
					SessionID:     ref.ID,
					SourcePath:    ref.Path,
					Workspace:     workspace,
					RawChars:      stats.RawChars,
					FilteredChars: stats.FilteredChars,
				},
				filteredText: stats.FilteredText,
			}, nil
		}
		if provider == providerCodex {
			return handoffInput{}, err
		}
	case providerAntigravity:
		return handoffInput{}, fmt.Errorf("antigravity provider is recognized but session parsing is not implemented yet")
	case providerClaudeCode:
		return resolveClaudeHandoffSession(prefix, store, reader)
	default:
		return handoffInput{}, fmt.Errorf("unknown provider %q", provider)
	}
	return resolveClaudeHandoffSession(prefix, store, reader)
}

func resolveClaudeHandoffSession(prefix string, store parser.Store, reader session.TranscriptReader) (handoffInput, error) {
	resolved, err := store.ResolveSession(prefix)
	if err != nil {
		return handoffInput{}, err
	}
	if resolved.Path == "" {
		return handoffInput{}, fmt.Errorf("transcript not found: %s", resolved.ID)
	}
	events, err := reader.ReadAll(resolved.Path)
	if err != nil {
		return handoffInput{}, err
	}
	stats := analyzer.ComputeStats(events)
	return handoffInput{
		info: handoff.SessionInfo{
			Provider:      session.ProviderClaudeCode,
			SessionID:     resolved.ID,
			SourcePath:    resolved.Path,
			Workspace:     filepath.Base(filepath.Dir(resolved.Path)),
			RawChars:      stats.RawChars,
			FilteredChars: stats.FilteredChars,
		},
		filteredText: stats.FilteredText,
	}, nil
}
