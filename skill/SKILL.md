---
name: sessions
description: |
  用 sessions CLI 讀取過去的 Claude Code session，而不是直接讀 JSONL 檔案。
  JSONL 原始檔動輒數萬行，會佔滿 context；sessions CLI 在 context 外完成過濾，
  只保留對話文字和 tool call 一行摘要。tool I/O 為主的 session 典型 reduction 80-88%，
  以大型 plan 文件或純對話為主者較低（約 40-65%）。
  使用者想回顧、引用、分析過去的對話時使用。
allowed-tools:
  - Bash
  - Read
---

# Session Reader

呼叫前先確保 PATH 包含 Go bin：`source ~/.zshrc` 或 `export PATH="$HOME/go/bin:$PATH"`。

若 `sessions` 未安裝：`go install github.com/Mapleeeeeeeeeee/cc-session-reader/cmd/sessions@latest`

## 選擇子命令

根據使用者的意圖選擇對應的子命令：

| 意圖 | 子命令 | 說明 |
|------|--------|------|
| 找到目標 session | `sessions list` | 列出最近的 session，支援 `-p` 過濾專案 |
| 回顧完整對話脈絡 | `sessions read <id>` | 對話全文 + 每個 tool call 壓成一行摘要 |
| 注入前次 session 為 context | `sessions context <id>` | 同 read 但格式更緊湊，帶 metadata header |
| 分析 token 節省效果 | `sessions stats <id>` | 各類別字元分佈和壓縮比 |
| 檢查過濾是否漏掉重要內容 | `sessions audit <id>` | 從被過濾的內容取樣檢視 |
| 展開特定 tool call 完整內容 | `sessions expand <id> <tool-id> [...]` | read 輸出的 #xxxx 就是 tool-id |

Session ID 支援 prefix match，前 8 碼通常就夠。各子命令的 flags 用 `sessions <cmd> --help` 查看。

## 常用 flags

- `-verbose-agents`（read/context）：完整保留 Agent subagent 回傳的分析結果，用於優化 skill/agent prompt 時開啟
- `-verbose-bash`（read/context）：完整顯示 Bash 工具的 stdout/stderr（預設摘要）
- `-verbose-thinking`（read/context）：顯示 assistant 的 thinking 區塊（預設隱藏）
- `-verbose-commands`（read/context）：展開 slash／bash 指令的完整輸出（預設只留 `[/cmd]`／`[!cmd]` marker、丟棄終端 UI 與 caveat 樣板）
- `-max-lines N`（read）：限制輸出行數，避免大 session 佔滿 context
- `-no-tokens`（stats）：跳過 token 計算，只看字元分佈

## 過濾邏輯

保留對話文字和 tool call 一行摘要；過濾 tool result 原始輸出、檔案內容、tool input JSON、system/noise messages，
以及 slash／bash 指令的終端輸出（壓成一行 marker）。
reduction 視 session 組成而定：tool I/O 為主者典型 80-88%，大型 plan 文件或純對話為主者較低（約 40-65%）。
