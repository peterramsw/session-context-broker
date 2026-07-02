package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/analyzer"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/codexcodec"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/config"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/distiller"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/handoff"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/redaction"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
)

func cmdHandoff(args []string, reader session.TranscriptReader) {
	exitOnError(runHandoff(args, os.Stdout, os.Stderr, parser.DefaultStore(), reader))
}

func runHandoff(args []string, out io.Writer, errOut io.Writer, store parser.Store, reader session.TranscriptReader) error {
	fs := flag.NewFlagSet("handoff", flag.ContinueOnError)
	fs.SetOutput(errOut)
	provider := fs.String("provider", providerAuto, "session provider: auto, claude_code, codex, antigravity")
	configPath := fs.String("config", "", "path to session-context config.json")
	storageRoot := fs.String("out", "", "override storage root for generated handoff artifacts")
	llmModeFlag := fs.String("llm", "auto", "Local LLM mode: auto, always, never")
	minFilteredCharsFlag := fs.Int("min-filtered-chars", -1, "override --llm auto threshold in redacted filtered chars")
	force := fs.Bool("force", false, "overwrite existing handoff artifacts")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("session_id is required")
	}
	llmMode, err := parseLLMMode(*llmModeFlag)
	if err != nil {
		return err
	}
	if *minFilteredCharsFlag < -1 {
		return fmt.Errorf("--min-filtered-chars must be >= 0")
	}

	cfg := config.LoadSessionContext()
	if *configPath != "" {
		cfg = config.LoadSessionContextFromPath(*configPath)
	}
	if *storageRoot != "" {
		cfg.StorageRoot = *storageRoot
	}
	minFilteredChars := cfg.LocalLLM.MinFilteredCharsOrDefault()
	if *minFilteredCharsFlag >= 0 {
		minFilteredChars = *minFilteredCharsFlag
	}

	input, err := resolveHandoffSession(fs.Arg(0), normalizeProvider(*provider), store, reader)
	if err != nil {
		return err
	}
	logUsageAsync("handoff", session.ShortID(input.info.SessionID, 8))

	redactedFiltered := redaction.RedactSecrets(input.filteredText)
	filteredPath, err := handoff.WriteFilteredArtifact(cfg.StorageRoot, input.info, redactedFiltered, *force)
	if err != nil {
		return err
	}
	decision := decideLLMUse(llmMode, len(redactedFiltered), minFilteredChars, cfg.LocalLLM.IsEnabled())
	if llmMode == llmModeAlways && !decision.UseLLM {
		return fmt.Errorf("Local LLM is not enabled; configure local_llm.enabled=true with base_url/model, or use --llm auto/never for filtered-only output")
	}
	if !decision.UseLLM {
		fmt.Fprintf(out, "Mode: filtered\n")
		fmt.Fprintf(out, "Provider: %s\n", input.info.Provider)
		fmt.Fprintf(out, "Session: %s\n", input.info.SessionID)
		fmt.Fprintf(out, "LLM policy: %s\n", llmMode)
		fmt.Fprintf(out, "LLM threshold: %d\n", minFilteredChars)
		fmt.Fprintf(out, "LLM decision: %s\n", decision.Reason)
		fmt.Fprintf(out, "Raw chars: %s\n", analyzer.FormatNumber(input.info.RawChars))
		fmt.Fprintf(out, "Filtered chars: %s\n", analyzer.FormatNumber(input.info.FilteredChars))
		fmt.Fprintf(out, "Redacted input chars: %s\n", analyzer.FormatNumber(len(redactedFiltered)))
		fmt.Fprintf(out, "Filtered output: %s\n", filteredPath)
		return nil
	}

	req := distiller.Request{
		Config:             cfg.LocalLLM,
		Session:            input.info,
		FilteredTranscript: redactedFiltered,
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
	fmt.Fprintf(out, "Mode: llm\n")
	fmt.Fprintf(out, "Provider: %s\n", input.info.Provider)
	fmt.Fprintf(out, "Session: %s\n", input.info.SessionID)
	fmt.Fprintf(out, "LLM policy: %s\n", llmMode)
	fmt.Fprintf(out, "LLM threshold: %d\n", minFilteredChars)
	fmt.Fprintf(out, "LLM decision: %s\n", decision.Reason)
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
	fmt.Fprintf(out, "Filtered output: %s\n", filteredPath)
	fmt.Fprintf(out, "Output: %s\n", dir)
	return nil
}

type llmMode string

const (
	llmModeAuto   llmMode = "auto"
	llmModeAlways llmMode = "always"
	llmModeNever  llmMode = "never"
)

type llmDecision struct {
	UseLLM bool
	Reason string
}

func parseLLMMode(value string) (llmMode, error) {
	switch llmMode(strings.ToLower(strings.TrimSpace(value))) {
	case llmModeAuto:
		return llmModeAuto, nil
	case llmModeAlways:
		return llmModeAlways, nil
	case llmModeNever:
		return llmModeNever, nil
	default:
		return "", fmt.Errorf("--llm must be auto, always, or never")
	}
}

func decideLLMUse(mode llmMode, redactedFilteredChars int, minFilteredChars int, localLLMEnabled bool) llmDecision {
	switch mode {
	case llmModeNever:
		return llmDecision{UseLLM: false, Reason: "--llm never requested"}
	case llmModeAlways:
		if !localLLMEnabled {
			return llmDecision{UseLLM: false, Reason: "--llm always requested, but Local LLM is not enabled"}
		}
		return llmDecision{UseLLM: true, Reason: "--llm always requested"}
	default:
		if redactedFilteredChars < minFilteredChars {
			return llmDecision{
				UseLLM: false,
				Reason: fmt.Sprintf("redacted filtered chars %d below threshold %d", redactedFilteredChars, minFilteredChars),
			}
		}
		if !localLLMEnabled {
			return llmDecision{
				UseLLM: false,
				Reason: fmt.Sprintf("redacted filtered chars %d meets threshold %d, but Local LLM is not enabled", redactedFilteredChars, minFilteredChars),
			}
		}
		return llmDecision{
			UseLLM: true,
			Reason: fmt.Sprintf("redacted filtered chars %d meets threshold %d", redactedFilteredChars, minFilteredChars),
		}
	}
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
