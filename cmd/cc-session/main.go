// Package main is the CLI entry point for the session context broker.
// Claude Code remains the default provider; Codex support is exposed through
// provider-aware commands as the fork grows into a cross-agent tool.
package main

import (
	"fmt"
	"os"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/claudecodec"
	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/tokens"
)

var version = "dev"

type countTokensFunc func(string) (int, error)

// countTokensFn is the token-counting backend used by runStats. It is a
// package-level seam so tests can substitute a deterministic offline stub
// (success or failure) without making real Anthropic API calls.
var countTokensFn countTokensFunc = tokens.CountTokensAPI

// newCountTokensFn builds a reusable token-counting backend for commands that
// count multiple inputs in one run.
var newCountTokensFn = func(model string) (countTokensFunc, error) {
	counter, err := tokens.NewCounter(model)
	if err != nil {
		return nil, err
	}
	return counter.Count, nil
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	defer waitUsageLog()

	reader := claudecodec.Codec{}

	subcommand := os.Args[1]
	switch subcommand {
	case "-h", "--help", "help":
		printUsage()
		return
	case "-v", "--version", "version":
		fmt.Printf("cc-session %s\n", version)
		return
	case "list":
		cmdList(os.Args[2:], reader)
	case "inspect":
		cmdInspect(os.Args[2:], reader)
	case "filter":
		cmdFilter(os.Args[2:], reader)
	case "handoff":
		cmdHandoff(os.Args[2:], reader)
	case "search":
		cmdSearch(os.Args[2:], reader)
	case "evidence":
		cmdEvidence(os.Args[2:], reader)
	case "verify-workspace":
		cmdVerifyWorkspace(os.Args[2:], reader)
	case "serve-mcp":
		cmdServeMCP(os.Args[2:], reader)
	case "read":
		cmdRead(os.Args[2:], reader)
	case "context":
		cmdContext(os.Args[2:], reader)
	case "stats":
		cmdStats(os.Args[2:], reader)
	case "audit":
		cmdAudit(os.Args[2:], reader)
	case "expand":
		cmdExpand(os.Args[2:], reader)
	case "usage":
		cmdUsage(os.Args[2:])
	case "inject":
		cmdInject(os.Args[2:], reader)
	case "benchmark":
		cmdBenchmark(os.Args[2:], reader)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: cc-session <command> [options]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  list      列出最近的 session")
	fmt.Fprintln(os.Stderr, "  inspect   顯示 session metadata / stats")
	fmt.Fprintln(os.Stderr, "  filter    輸出 deterministic filtered transcript")
	fmt.Fprintln(os.Stderr, "  handoff   產生 Local LLM handoff artifact")
	fmt.Fprintln(os.Stderr, "  search    搜尋 session evidence summaries")
	fmt.Fprintln(os.Stderr, "  evidence  依 evidence ID 展開 redacted source bytes")
	fmt.Fprintln(os.Stderr, "  verify-workspace  唯讀檢查允許 root 內的 git workspace")
	fmt.Fprintln(os.Stderr, "  serve-mcp 啟動 stdio MCP server")
	fmt.Fprintln(os.Stderr, "  read      完整對話 + tool call 一行摘要")
	fmt.Fprintln(os.Stderr, "  context   精簡注入格式（帶 metadata header）")
	fmt.Fprintln(os.Stderr, "  stats     字元與 token 分佈統計")
	fmt.Fprintln(os.Stderr, "  audit     檢視被過濾的內容取樣")
	fmt.Fprintln(os.Stderr, "  expand    展開特定 tool call 完整內容")
	fmt.Fprintln(os.Stderr, "  usage     CLI 使用紀錄")
	fmt.Fprintln(os.Stderr, "  inject    分頁注入 session 到 context")
	fmt.Fprintln(os.Stderr, "  benchmark 掃描近期 session，計算壓縮率與成本比較")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Run 'cc-session <command> -h' for command-specific flags.")
}
