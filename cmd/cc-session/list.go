package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/codexcodec"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/parser"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/session"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/tracker"
)

func cmdList(args []string, scanner session.HeaderScanner) {
	exitOnError(runList(args, os.Stdout, os.Stderr, parser.DefaultStoreWith(scanner)))
}

func runList(args []string, out io.Writer, errOut io.Writer, store parser.Store) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(errOut)
	limit := fs.Int("n", 20, "max sessions to display")
	project := fs.String("p", "", "filter by project name (case-insensitive)")
	provider := fs.String("provider", providerClaudeCode, "session provider: claude_code, codex, antigravity, all")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *limit < 1 {
		return fmt.Errorf("-n must be a positive integer")
	}
	logUsageAsync("list", "")

	switch normalizeProvider(*provider) {
	case providerClaudeCode:
		// Existing behavior below.
	case providerCodex:
		return runCodexList(*limit, *project, out)
	case providerAntigravity:
		return fmt.Errorf("antigravity provider is recognized but session parsing is not implemented yet")
	case providerAll:
		if err := runList(argsWithoutProvider(args), out, errOut, store); err != nil {
			return err
		}
		return runCodexList(*limit, *project, out)
	default:
		return fmt.Errorf("unknown provider %q", *provider)
	}

	projectFilter := ""
	if *project != "" {
		projectFilter = strings.ToLower(*project)
	}

	entries, warnings := store.ListAllSessions()
	for _, w := range warnings {
		fmt.Fprintln(errOut, w)
	}
	callerIDs := tracker.CallerSessionIDs()

	printed := 0
	for _, entry := range entries {
		if printed >= *limit {
			break
		}

		projectName := "?"
		if entry.ProjectPath != "" {
			projectName = filepath.Base(entry.ProjectPath)
		}

		if projectFilter != "" && !strings.Contains(strings.ToLower(projectName), projectFilter) {
			continue
		}

		dateStr := "??-??"
		if entry.StartTime != "" {
			dateStr = parser.FormatTimestamp(entry.StartTime)
		}

		countStr := fmt.Sprintf("u:%d a:%d", entry.UserMessageCount, entry.AssistantMessageCount)
		if entry.UserMessageCount == 0 && entry.AssistantMessageCount == 0 {
			countStr = "       "
		}
		refMarker := ""
		if callerIDs[entry.SessionID] {
			refMarker = " [refs]"
		}
		fmt.Fprintf(out, "%s  %s  %-20s  %3dm  %s%s  %s\n",
			entry.SessionID, dateStr, projectName, entry.DurationMinutes, countStr, refMarker, entry.FirstPrompt)
		printed++
	}

	if printed == 0 {
		fmt.Fprintln(out, "No sessions found.")
	}
	return nil
}

func runCodexList(limit int, project string, out io.Writer) error {
	codec := codexcodec.Codec{}
	refs, err := codec.Discover()
	if err != nil {
		return err
	}
	projectFilter := strings.ToLower(project)
	printed := 0
	for _, ref := range refs {
		if printed >= limit {
			break
		}
		projectName := "?"
		if ref.ProjectPath != "" {
			projectName = filepath.Base(ref.ProjectPath)
		}
		if projectFilter != "" && !strings.Contains(strings.ToLower(projectName), projectFilter) {
			continue
		}
		dateStr := "??-??"
		if ref.StartTime != "" {
			dateStr = parser.FormatTimestamp(ref.StartTime)
		}
		fmt.Fprintf(out, "%s  %s  %-20s  %3s  %-11s  %s\n",
			ref.ID, dateStr, projectName, "", "[codex]", ref.FirstPrompt)
		printed++
	}
	if printed == 0 {
		fmt.Fprintln(out, "No sessions found.")
	}
	return nil
}

func argsWithoutProvider(args []string) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		if args[i] == "-provider" || args[i] == "--provider" {
			i++
			continue
		}
		if strings.HasPrefix(args[i], "-provider=") || strings.HasPrefix(args[i], "--provider=") {
			continue
		}
		out = append(out, args[i])
	}
	return out
}
