#!/usr/bin/env bash
set -euo pipefail

REPO="peterramsw/session-context-broker"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
SKILL_DIR="$HOME/.claude/skills/cc-session"
SKILL_URL="https://raw.githubusercontent.com/${REPO}/main/SKILL.md"
SKIP_SKILL=0
CLIENTS=""

# ── arg parsing ──────────────────────────────────────────────────────────────

for arg in "$@"; do
  case "$arg" in
    --no-skill) SKIP_SKILL=1; CLIENTS="none" ;;
    --clients=*) CLIENTS="${arg#--clients=}" ;;
    --help|-h)
      echo "Usage: install.sh [--no-skill] [--clients all|none|claude,codex,antigravity]"
      echo "  --no-skill  Skip installing the Claude Code skill"
      echo "  --clients   Select client integrations non-interactively"
      exit 0
      ;;
    *)
      echo "Unknown option: $arg" >&2
      exit 1
      ;;
  esac
done

# ── helpers ───────────────────────────────────────────────────────────────────

is_tty() { [ -t 0 ]; }

detect_shell_rc() {
  case "${SHELL:-}" in
    */zsh)  echo "$HOME/.zshrc" ;;
    */bash) echo "$HOME/.bashrc" ;;
    *)      echo "$HOME/.profile" ;;
  esac
}

# ── platform detection ────────────────────────────────────────────────────────

detect_platform() {
  local os arch

  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    *)
      echo "Unsupported OS: $(uname -s)" >&2
      exit 1
      ;;
  esac

  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)
      echo "Unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac

  echo "${os}_${arch}"
}

# ── latest version lookup ─────────────────────────────────────────────────────

fetch_latest_version() {
  local api_url="https://api.github.com/repos/${REPO}/releases/latest"
  local version

  if command -v curl &>/dev/null; then
    version=$(curl -fsSL "$api_url" | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
  elif command -v wget &>/dev/null; then
    version=$(wget -qO- "$api_url" | grep '"tag_name"' | sed 's/.*"tag_name": *"\(.*\)".*/\1/')
  else
    echo "Neither curl nor wget found. Please install one." >&2
    exit 1
  fi

  if [ -z "$version" ]; then
    echo "Failed to fetch latest release version." >&2
    exit 1
  fi

  echo "$version"
}

# ── binary download & install ─────────────────────────────────────────────────

TMPDIR_CLEANUP=""
trap '[ -n "$TMPDIR_CLEANUP" ] && rm -rf "$TMPDIR_CLEANUP"' EXIT

# stop_running_instances stops a cc-session running from the install path so the
# upgraded binary is actually used — a running MCP server keeps serving the old
# binary until it is restarted. Scoped strictly to INSTALL_DIR so unrelated
# processes are never touched.
stop_running_instances() {
  command -v pgrep >/dev/null 2>&1 || return 0
  local pids
  pids=$(pgrep -f "$INSTALL_DIR/cc-session" 2>/dev/null || true)
  if [ -n "$pids" ]; then
    echo "Stopping running cc-session so it reloads the new binary..."
    # shellcheck disable=SC2086
    kill $pids 2>/dev/null || true
    sleep 1
  fi
}

install_binary() {
  local version="$1"
  local platform="$2"
  local version_bare="${version#v}"

  local download_url="https://github.com/${REPO}/releases/download/${version}/session-context-broker_${version_bare}_${platform}.tar.gz"
  TMPDIR_CLEANUP=$(mktemp -d)

  echo "Downloading cc-session ${version} for ${platform}..."

  if command -v curl &>/dev/null; then
    curl -fsSL "$download_url" -o "$TMPDIR_CLEANUP/archive.tar.gz"
  else
    wget -qO "$TMPDIR_CLEANUP/archive.tar.gz" "$download_url"
  fi

  tar -xzf "$TMPDIR_CLEANUP/archive.tar.gz" -C "$TMPDIR_CLEANUP"

  mkdir -p "$INSTALL_DIR"
  stop_running_instances
  mv "$TMPDIR_CLEANUP/cc-session" "$INSTALL_DIR/cc-session"
  chmod +x "$INSTALL_DIR/cc-session"

  # Verify the swap actually took effect instead of trusting `mv`'s exit code
  # alone — this is the safety net that catches a stale binary left in place
  # by any failure mode upstream (e.g. a lock that mv silently won races with).
  local installed_version
  installed_version=$("$INSTALL_DIR/cc-session" --version 2>&1 || true)
  case "$installed_version" in
    *"$version_bare"*) ;;
    *)
      echo "Error: installed binary reports '$installed_version', expected version $version_bare. Install did not take effect — re-run the installer." >&2
      exit 1
      ;;
  esac

  echo "Installed cc-session ${version} to $INSTALL_DIR/cc-session"
  echo "If an agent already had the MCP server open, restart Claude Code / Codex / Antigravity to load the new version."
}

# ── PATH check ────────────────────────────────────────────────────────────────

check_path() {
  if echo ":${PATH}:" | grep -q ":${INSTALL_DIR}:"; then
    return
  fi

  local export_line="export PATH=\"\$PATH:$INSTALL_DIR\""

  if ! is_tty; then
    echo ""
    echo "Warning: $INSTALL_DIR is not in your PATH."
    echo "Add it manually: $export_line"
    return
  fi

  echo ""
  echo "Warning: $INSTALL_DIR is not in your PATH."
  local shell_rc
  shell_rc=$(detect_shell_rc)
  printf "Add %s to PATH in %s? [y/N] " "$INSTALL_DIR" "$shell_rc"
  read -r answer
  if [[ "$answer" =~ ^[Yy]$ ]]; then
    echo "" >> "$shell_rc"
    echo "# Added by cc-session-reader installer" >> "$shell_rc"
    echo "$export_line" >> "$shell_rc"
    echo "Added to $shell_rc. Run: source $shell_rc"
  fi
}

# ── skill install ─────────────────────────────────────────────────────────────

install_skill() {
  if [ "$SKIP_SKILL" -eq 1 ]; then
    return
  fi

  local selected="$CLIENTS"
  if [ -z "$selected" ]; then
    if is_tty; then
      echo "Select client integrations to install:"
      print_client_status "claude" "$HOME/.claude/skills/cc-session/SKILL.md"
      print_client_status "codex" "$HOME/.codex/skills/cc-session/SKILL.md"
      print_client_status "antigravity" "$HOME/.gemini/antigravity/skills/cc-session/SKILL.md"
      printf "Clients [claude] (all|none|claude,codex,antigravity): "
      read -r selected
      selected="${selected:-claude}"
    else
      selected="claude"
    fi
  fi
  case "$selected" in
    none) return ;;
    all) selected="claude,codex,antigravity" ;;
  esac

  IFS=',' read -r -a clients <<< "$selected"
  for client in "${clients[@]}"; do
    case "$(echo "$client" | tr '[:upper:]' '[:lower:]' | xargs)" in
      claude|claude_code|claude-code)
        install_client_skill "claude-code" "$HOME/.claude/skills/cc-session" "Claude Code"
        ;;
      codex)
        install_client_skill "codex" "$HOME/.codex/skills/cc-session" "Codex"
        register_mcp_codex
        ;;
      antigravity|angravity)
        install_client_skill "antigravity" "$HOME/.gemini/antigravity/skills/cc-session" "Google Antigravity standalone app"
        register_mcp_antigravity
        ;;
      "")
        ;;
      *)
        echo "Unknown client: $client" >&2
        exit 1
        ;;
    esac
  done
}

# register_mcp_* functions wire cc-session into each client's own MCP config so
# a fresh install is actually usable as an MCP server, not just a skill. They
# are idempotent (skip if an entry already exists) and never touch other
# entries in the file.

register_mcp_codex() {
  local config="$HOME/.codex/config.toml"
  mkdir -p "$(dirname "$config")"
  touch "$config"
  if grep -q '^\[mcp_servers\.cc-session\]' "$config" 2>/dev/null; then
    echo "Codex MCP: cc-session already registered."
    return
  fi
  {
    echo ""
    echo "[mcp_servers.cc-session]"
    echo "command = \"$INSTALL_DIR/cc-session\""
    echo "args = [\"serve-mcp\"]"
  } >> "$config"
  echo "Registered cc-session as a Codex MCP server in $config"
}

register_mcp_antigravity() {
  local config="$HOME/.gemini/antigravity/mcp_config.json"
  mkdir -p "$(dirname "$config")"
  if ! command -v python3 &>/dev/null; then
    echo "Warning: python3 not found; add cc-session to $config manually:" >&2
    echo "  \"cc-session\": {\"command\": \"$INSTALL_DIR/cc-session\", \"args\": [\"serve-mcp\"]}" >&2
    return
  fi
  python3 - "$config" "$INSTALL_DIR/cc-session" <<'PYEOF'
import json, sys

path, cmd = sys.argv[1], sys.argv[2]
try:
    with open(path) as f:
        content = f.read().strip()
    data = json.loads(content) if content else {}
except FileNotFoundError:
    data = {}
except json.JSONDecodeError:
    print(f"Warning: {path} is not valid JSON; skipping cc-session registration. Add it manually.", file=sys.stderr)
    sys.exit(0)

servers = data.setdefault("mcpServers", {})
if "cc-session" in servers:
    print("Antigravity MCP: cc-session already registered.")
else:
    servers["cc-session"] = {"command": cmd, "args": ["serve-mcp"]}
    with open(path, "w") as f:
        json.dump(data, f, indent=2)
    print(f"Registered cc-session as an Antigravity MCP server in {path}")
PYEOF
}

print_client_status() {
  local name="$1"
  local path="$2"
  if [ -f "$path" ]; then
    echo "  [x] $name"
  else
    echo "  [ ] $name"
  fi
}

download_raw() {
  local url="$1"
  local dst="$2"
  if command -v curl &>/dev/null; then
    curl -fsSL "$url" -o "$dst"
  else
    wget -qO "$dst" "$url"
  fi
}

install_client_skill() {
  local source="$1"
  local target="$2"
  local label="$3"
  local base="https://raw.githubusercontent.com/${REPO}/main/skills"
  mkdir -p "$target/common"
  echo "Installing $label skill to $target..."
  download_raw "$base/$source/cc-session/SKILL.md" "$target/SKILL.md"
  download_raw "$base/common/resume-session.md" "$target/common/resume-session.md"
  download_raw "$base/common/close-session.md" "$target/common/close-session.md"
  download_raw "$base/common/review-history.md" "$target/common/review-history.md"

  echo "$label integration installed."
}

# ── main ──────────────────────────────────────────────────────────────────────

print_next_steps() {
  echo ""
  echo "── Getting started ────────────────────────────────────────────────"
  echo "  cc-session list          # 列出最近的 session"
  echo "  cc-session read <id>     # 讀取對話內容"
  echo "  /cc-session              # 在 Claude Code 中使用 (需已安裝 Skill)"
  echo ""
  echo "── Token counting (optional) ──────────────────────────────────────"
  echo "  For precise token counts in 'cc-session stats', create:"
  echo "  $SKILL_DIR/config.json"
  echo ""
  echo '  {"anthropic_api_key_file": "<path-to-your-api-key-file>"}'
  echo ""
}

main() {
  local version platform
  version=$(fetch_latest_version)
  platform=$(detect_platform)

  install_binary "$version" "$platform"
  check_path
  install_skill
  print_next_steps
}

main
