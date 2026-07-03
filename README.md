# session-context-broker

**繁體中文** ｜ [English](README.en.md)

> **本專案 fork 自 [`Mapleeeeeeeeeee/cc-session-reader`](https://github.com/Mapleeeeeeeeeee/cc-session-reader)，授權延用上游的 Apache License 2.0。**
> 上游是一個純靜態（不使用 LLM）的 **Claude Code** session 讀取工具，把 transcript 中的大量 tool 雜訊壓縮掉、保留對話，讓過去的 session 能用很少的 token 重新載入。本 fork 保留這個核心，並把它擴充成跨 agent 的 **session context broker**。

## 這個 fork 在上游之上加了什麼

- **多來源 session**：除了 Claude Code，還支援 **Codex CLI** 與 **Google Antigravity 2.0** 的 session，全部收斂到同一套 normalized provider adapter（上游只支援 Claude Code）。
- **可選的本地 LLM handoff 蒸餾**：透過 OpenAI-compatible 的本地 endpoint，把過濾後的 transcript 蒸餾成結構化、可回查證據的 `handoff.json`。**預設關閉**——沒有本地 LLM 的人一樣能用過濾與 evidence 功能。
- **Evidence store**：過濾輸出、evidence 索引、handoff 產物都持久化在 `storage_root`，可依 evidence ID 按需展開。
- **MCP server**：`cc-session serve-mcp` 把 broker 以工具形式開放給 Claude Code / Codex / Antigravity。
- **跨 agent skills**：可安裝的 resume / close / review-history 工作流程。

上游既有 CLI 行為（`list`、`read`、`context`、`inject`、`stats`、`expand`、`audit`）完整保留。

## 機制（Pipeline）

核心流程是「先靜態過濾，再選擇性交給本地 LLM」，原始 session 永遠不被修改：

```
原始 session（Claude Code / Codex / Antigravity）
  → deterministic filter   ← 壓掉 tool 雜訊，保留對話與風險訊號（error/rollback/exit code…）
  → secret redaction       ← 預設遮罩 API key、token、密碼等
  → evidence index         ← 每個被壓縮的片段給一個可回查的 evidence_id
  → [可選] 本地 LLM 蒸餾    ← 產出結構化 handoff.json（objective / decisions / next_actions…）
  → schema + evidence 驗證  ← 沒有證據的聲明會被降級，不會被當成「已確認」
  → MCP / Skill            ← 提供給新的 session 接手
```

兩個層次的價值要分清楚：

- **靜態過濾（無 LLM）**：以 tool I/O 為主的 session 典型可壓縮 **80–88%**（純對話或大型 plan 文件較低）。這一層**沒有幻覺風險**，是省 token 的主力。
- **本地 LLM 蒸餾（可選）**：不是為了再省 token，而是把長 session 整理成可導航的結構（目標、下一步、待重驗清單）。適合「接手工具活動密集的工程 session」。它是 **derived artifact，不是真相來源**；沒有證據的聲明一律降級到 `claims_requiring_reverification`。

> 換句話說：省 token 靠過濾層就達成了；本地 LLM 是「導航」增益，需要時才開。

### 什麼時候會被呼叫？

**沒有背景自動執行、沒有計時器**——安裝完之後，若什麼都不做，這套工具完全不會動，也不會影響你目前這個對話的 token 用量。它只在下列情況才會被呼叫：

- **意圖觸發（預設）**：Skill 的 `description` 讓 Claude Code / Codex / Antigravity 自動判斷。當你自然地說「接續上次那個 session」「上次做到哪」「幫我收尾這個對話」，agent 會自己決定呼叫 `cc-session`，不需要背指令。
- **手動觸發**：你也可以直接明講「跑 `cc-session list`」或請 agent 呼叫某個 MCP tool。

它管的是**跨 session**（把舊對話便宜地載入新對話），不是壓縮你正在進行中的這個對話——這個對話的 token 用量不受它影響。

## 安裝

### 一鍵安裝

安裝腳本會下載對應平台的 binary，並可同時安裝 Claude Code / Codex / Antigravity 的 skill。

**macOS / Linux**

```bash
curl -fsSL https://raw.githubusercontent.com/peterramsw/session-context-broker/main/install.sh | bash -s -- --clients claude,codex,antigravity
```

**Windows PowerShell**

```powershell
irm https://raw.githubusercontent.com/peterramsw/session-context-broker/main/install.ps1 | iex
```

### 選擇要安裝哪些 client

`--clients` 支援 `all`、`none`，或逗號分隔的 `claude,codex,antigravity`。互動模式會把「已安裝」的 client 打勾顯示。

```bash
./install.sh --clients all                    # 三個都裝
./install.sh --clients claude                 # 只裝 Claude Code
./install.sh --clients codex,antigravity      # 只裝 Codex + Antigravity
./install.sh --no-skill                        # 只裝 binary，不裝任何 skill
```

```powershell
.\install.ps1 -Clients all
.\install.ps1 -Clients claude
.\install.ps1 -Clients codex,antigravity
```

各 client 的 skill 安裝位置：

| Client | Skill 路徑 |
|---|---|
| Claude Code | `~/.claude/skills/cc-session` |
| Codex | `~/.codex/skills/cc-session` |
| Google Antigravity 2.0 | `~/.gemini/antigravity/skills/cc-session` |

### 其他安裝方式

- **Releases**：從 [GitHub Releases](https://github.com/peterramsw/session-context-broker/releases) 下載對應平台壓縮檔，解壓後把 `cc-session` 放進 PATH。
- **從原始碼建置**：`git clone` 後 `go build ./cmd/cc-session`。（注意：`go install` 目前會裝到**上游**版本，因為 module path 仍沿用上游 `github.com/Mapleeeeeeeeeee/cc-session-reader`。）

## 使用

### CLI 子命令

| 命令 | 用途 |
|---|---|
| `list` | 依 provider 列出 session |
| `inspect` | 顯示 session metadata 與統計 |
| `filter` | 印出靜態過濾後的 transcript |
| `handoff` | 寫出 filtered/evidence 產物，並可選擇性做本地 LLM handoff |
| `search` | 搜尋 evidence 摘要 |
| `evidence` | 依 evidence ID 展開（預設遮罩） |
| `verify-workspace` | 在允許的 root 內做唯讀 git 檢查 |
| `serve-mcp` | 啟動 stdio MCP server |
| `read`、`context`、`stats`、`audit`、`expand`、`inject`、`benchmark` | 保留的上游命令 |

範例：

```bash
cc-session list --provider all -n 10
cc-session filter --provider codex <session-id>

# 沒有本地 LLM：只產出過濾 + evidence 產物
cc-session handoff --provider antigravity --llm never <session-id>

# 有本地 LLM：達到門檻才蒸餾（auto），或強制蒸餾（always）
cc-session handoff --provider codex --llm auto <session-id>

cc-session serve-mcp --config ~/.session-context/config.json
```

### 接上 MCP

任一支援 MCP 的 client 都是啟動同一個 stdio server：`cc-session serve-mcp`。以 Claude Code 的專案設定 `.mcp.json` 為例：

```json
{
  "mcpServers": {
    "cc-session": {
      "command": "cc-session",
      "args": ["serve-mcp"]
    }
  }
}
```

Codex 與 Antigravity 依各自的 MCP 設定格式指向同一個 `cc-session serve-mcp` 命令即可。提供的工具（`list_sessions`、`inspect_session`、`filter_session`、`create_handoff`、`get_handoff`、`search_session`、`expand_evidence`、`compare_context_size`、`verify_workspace`）見 [docs/mcp-tools.md](docs/mcp-tools.md)。

## 設定

預設路徑 `~/.session-context/config.json`，可用 `SESSION_CONTEXT_CONFIG` 覆蓋。**此檔為選擇性**——不設定時仍可 list/inspect/filter/search，只是不啟用本地 LLM。

```json
{
  "session_sources": {
    "claude_code": {"roots": ["~/.claude/projects"]},
    "codex": {"roots": ["~/.codex/sessions"]},
    "antigravity": {"roots": ["~/.gemini/antigravity/brain"]}
  },
  "storage_root": "~/.session-context",
  "allowed_workspace_roots": ["~/work"],
  "local_llm": {
    "enabled": false,
    "base_url": "http://127.0.0.1:8000/v1",
    "api_key": "",
    "model": "Qwen3.6-35B-A3B",
    "max_context": 32000,
    "max_output_tokens": 4096,
    "timeout_seconds": 120,
    "min_filtered_chars": 8000,
    "temperature": 0,
    "top_p": 0.95,
    "top_k": 20
  }
}
```

環境變數可覆蓋設定，包含 `SESSION_CONTEXT_STORAGE_ROOT`、`SESSION_CONTEXT_LOCAL_LLM_ENABLED`、`LOCAL_LLM_BASE_URL`、`LOCAL_LLM_API_KEY`、`LOCAL_LLM_MODEL`、`LOCAL_LLM_MAX_CONTEXT`、`LOCAL_LLM_MAX_OUTPUT_TOKENS`、`LOCAL_LLM_TIMEOUT_SECONDS`、`LOCAL_LLM_MIN_FILTERED_CHARS`、`LOCAL_LLM_TEMPERATURE`、`LOCAL_LLM_TOP_P`、`LOCAL_LLM_TOP_K`。

## 產出 artifacts

`cc-session handoff` 寫在 `storage_root/<provider>/<session-id>/` 下：

- `manifest.json`、`normalized.jsonl`、`filtered.jsonl`、`filtered.md`、`evidence-index.json`
- `handoff.json` 與 `handoff.md`（僅在使用本地 LLM 時產生）

過濾產物與 evidence 展開預設遮罩敏感值；**原始 session 檔案永不修改**。

## 文件

- [Architecture](docs/architecture.md)
- [Session Providers](docs/session-provider.md)
- [Normalized Event Schema](docs/normalized-event-schema.md)
- [Handoff Schema](docs/handoff-schema.md)
- [Local LLM Distillation](docs/local-llm-distillation.md)
- [MCP Tools](docs/mcp-tools.md)
- [Skills](docs/skills.md)
- [Security](docs/security.md)
- [Upstream Sync](docs/upstream-sync.md)

## 授權

Apache License 2.0，**延用上游** `Mapleeeeeeeeeee/cc-session-reader`。`LICENSE` 檔維持不變；本 fork 新增的 Codex/Antigravity 支援、本地 LLM handoff、MCP 與 skills 同樣以 Apache-2.0 釋出。詳見 [LICENSE](LICENSE)。
