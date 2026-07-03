#Requires -Version 5.1
[CmdletBinding()]
param(
    [switch]$NoSkill,
    [string]$Clients = ""
)

$ErrorActionPreference = 'Stop'

function Read-HostOrDefault {
    param([string]$Prompt, [string]$Default)
    try {
        $result = Read-Host $Prompt
        if ([string]::IsNullOrEmpty($result)) { return $Default }
        return $result
    } catch {
        Write-Host "(non-interactive: using default '$Default')"
        return $Default
    }
}

$Repo      = "peterramsw/session-context-broker"
$InstallDir = Join-Path $env:LOCALAPPDATA "cc-session"
$SkillDir  = Join-Path $HOME ".claude\skills\cc-session"
$SkillUrl  = "https://raw.githubusercontent.com/$Repo/main/SKILL.md"

# ── architecture detection ────────────────────────────────────────────────────

function Get-Architecture {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        'X64'   { return 'amd64' }
        'Arm64' { return 'arm64' }
        default {
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}

# ── latest version lookup ─────────────────────────────────────────────────────

function Get-LatestVersion {
    $apiUrl = "https://api.github.com/repos/$Repo/releases/latest"
    try {
        $response = Invoke-RestMethod -Uri $apiUrl -UseBasicParsing
        $version = $response.tag_name
        if (-not $version) {
            Write-Error "Failed to parse release version from GitHub API."
            exit 1
        }
        return $version
    } catch {
        Write-Error "Failed to fetch latest release: $_"
        exit 1
    }
}

# ── binary download & install ─────────────────────────────────────────────────

# Stop a cc-session process running from the install path. On Windows a running
# exe is locked, so an upgrade cannot overwrite it; and even where it could, the
# already-running MCP server keeps serving the old binary until it is restarted.
# Match strictly on the executable path so we never touch an unrelated process.
# Returns once the file is actually unlocked (polls up to ~5s), since a killed
# process does not always release its file handle immediately on Windows.
function Stop-RunningInstances {
    $exeDst = Join-Path $InstallDir "cc-session.exe"
    $procs = @(Get-CimInstance Win32_Process -Filter "Name = 'cc-session.exe'" -ErrorAction SilentlyContinue |
        Where-Object { $_.ExecutablePath -and ($_.ExecutablePath -ieq $exeDst) })
    foreach ($p in $procs) {
        Write-Host "Stopping running cc-session (PID $($p.ProcessId)) so the binary can be replaced..."
        Stop-Process -Id $p.ProcessId -Force -ErrorAction SilentlyContinue
    }
    if (-not (Test-Path $exeDst)) { return }
    for ($i = 0; $i -lt 25; $i++) {
        try {
            $stream = [System.IO.File]::Open($exeDst, 'Open', 'ReadWrite', 'None')
            $stream.Close()
            return
        } catch {
            Start-Sleep -Milliseconds 200
        }
    }
    Write-Warning "cc-session.exe still appears locked after waiting; the install may fail to replace it."
}

function Install-Binary {
    param([string]$Version, [string]$Arch)

    $versionBare = $Version.TrimStart('v')
    $zipName     = "session-context-broker_${versionBare}_windows_${Arch}.zip"
    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/$zipName"
    $tmpDir      = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())

    Write-Host "Downloading cc-session $Version for windows/$Arch..."

    try {
        New-Item -ItemType Directory -Path $tmpDir | Out-Null
        $zipPath = Join-Path $tmpDir $zipName

        Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing -ErrorAction Stop
        Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force -ErrorAction Stop

        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -ErrorAction Stop | Out-Null
        }

        $exeSrc = Join-Path $tmpDir "cc-session.exe"
        $exeDst = Join-Path $InstallDir "cc-session.exe"
        if (-not (Test-Path $exeSrc)) {
            throw "downloaded archive did not contain cc-session.exe"
        }
        Stop-RunningInstances

        # Move-Item can still hit a transient lock right after Stop-Process even
        # after Stop-RunningInstances' own poll returns; retry briefly rather
        # than letting one failure abort or (worse) silently leave the old binary.
        $moved = $false
        for ($i = 0; $i -lt 10 -and -not $moved; $i++) {
            try {
                Move-Item -Path $exeSrc -Destination $exeDst -Force -ErrorAction Stop
                $moved = $true
            } catch {
                Start-Sleep -Milliseconds 300
            }
        }
        if (-not $moved) {
            throw "could not replace $exeDst (file still locked). Close any app using cc-session and re-run the installer."
        }

        # Verify the swap actually took effect instead of trusting Move-Item's
        # reported success — this is what catches a silent installer failure.
        $installedVersion = & $exeDst --version 2>&1
        if ($LASTEXITCODE -ne 0 -or -not ($installedVersion -match [regex]::Escape($versionBare))) {
            throw "installed binary reports '$installedVersion', expected version $versionBare. Install did not take effect — re-run the installer."
        }

        Write-Host "Installed cc-session $Version to $exeDst"
        Write-Host "If an agent already had the MCP server open, restart Claude Code / Codex / Antigravity to load the new version."
    } catch {
        Write-Error "cc-session install failed: $_"
        exit 1
    } finally {
        Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# ── PATH check ────────────────────────────────────────────────────────────────

function Update-UserPath {
    $currentPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
    $dirs = $currentPath -split ';' | Where-Object { $_ -ne '' }

    if ($dirs -contains $InstallDir) {
        return
    }

    Write-Host ""
    Write-Host "Warning: $InstallDir is not in your PATH."

    if (-not [Environment]::UserInteractive) {
        Write-Host "Add it manually to your user PATH."
        return
    }

    $answer = Read-HostOrDefault -Prompt "Add $InstallDir to user PATH? [Y/n]" -Default "Y"
    if ($answer -match '^[Yy]$') {
        $newPath = ($dirs + $InstallDir) -join ';'
        [Environment]::SetEnvironmentVariable('PATH', $newPath, 'User')
        Write-Host "Added to user PATH. Restart your terminal to apply."
    }
}

# ── skill install ─────────────────────────────────────────────────────────────

function Install-Skill {
    if ($NoSkill) { $script:Clients = "none" }

    $selected = $Clients
    if ([string]::IsNullOrWhiteSpace($selected)) {
        if ([Environment]::UserInteractive) {
            Write-Host "Select client integrations to install:"
            Show-ClientStatus -Name "claude" -Path (Join-Path $HOME ".claude\skills\cc-session\SKILL.md")
            Show-ClientStatus -Name "codex" -Path (Join-Path $HOME ".codex\skills\cc-session\SKILL.md")
            Show-ClientStatus -Name "antigravity" -Path (Join-Path $HOME ".gemini\antigravity\skills\cc-session\SKILL.md")
            $selected = Read-HostOrDefault -Prompt "Clients [claude] (all|none|claude,codex,antigravity)" -Default "claude"
        } else {
            $selected = "claude"
        }
    }
    if ($selected -eq "none") { return }
    if ($selected -eq "all") { $selected = "claude,codex,antigravity" }

    foreach ($client in ($selected -split ',')) {
        switch ($client.Trim().ToLowerInvariant()) {
            { $_ -in @("claude", "claude_code", "claude-code") } {
                Install-ClientSkill -Source "claude-code" -Target (Join-Path $HOME ".claude\skills\cc-session") -Label "Claude Code"
            }
            "codex" {
                Install-ClientSkill -Source "codex" -Target (Join-Path $HOME ".codex\skills\cc-session") -Label "Codex"
                Register-McpCodex
            }
            { $_ -in @("antigravity", "angravity") } {
                Install-ClientSkill -Source "antigravity" -Target (Join-Path $HOME ".gemini\antigravity\skills\cc-session") -Label "Google Antigravity standalone app"
                Register-McpAntigravity
            }
            "" {}
            default {
                Write-Error "Unknown client: $client"
                exit 1
            }
        }
    }
}

# register_mcp_* functions wire cc-session into each client's own MCP config so
# a fresh install is actually usable as an MCP server, not just a skill. They
# are idempotent (skip if an entry already exists) and never touch other
# entries in the file.

function Register-McpCodex {
    $config = Join-Path $HOME ".codex\config.toml"
    $dir = Split-Path $config -Parent
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
    if (-not (Test-Path $config)) { New-Item -ItemType File -Path $config -Force | Out-Null }

    $content = Get-Content -Raw -Path $config -ErrorAction SilentlyContinue
    if ($content -match '(?m)^\[mcp_servers\.cc-session\]') {
        Write-Host "Codex MCP: cc-session already registered."
        return
    }

    $exePath = Join-Path $InstallDir "cc-session.exe"
    # TOML literal string (single quotes) so Windows backslashes need no escaping.
    $block = @"

[mcp_servers.cc-session]
command = '$exePath'
args = ["serve-mcp"]
"@
    Add-Content -Path $config -Value $block
    Write-Host "Registered cc-session as a Codex MCP server in $config"
}

function Register-McpAntigravity {
    $config = Join-Path $HOME ".gemini\antigravity\mcp_config.json"
    $dir = Split-Path $config -Parent
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }

    $data = $null
    if (Test-Path $config) {
        try {
            $data = Get-Content -Raw -Path $config | ConvertFrom-Json -AsHashtable
        } catch {
            Write-Warning "$config is not valid JSON; skipping cc-session registration. Add it manually."
            return
        }
    }
    if (-not $data) { $data = @{} }
    if (-not $data.ContainsKey('mcpServers')) { $data['mcpServers'] = @{} }
    if ($data['mcpServers'].ContainsKey('cc-session')) {
        Write-Host "Antigravity MCP: cc-session already registered."
        return
    }

    $exePath = Join-Path $InstallDir "cc-session.exe"
    $data['mcpServers']['cc-session'] = @{ command = $exePath; args = @("serve-mcp") }
    ($data | ConvertTo-Json -Depth 10) | Set-Content -Path $config
    Write-Host "Registered cc-session as an Antigravity MCP server in $config"
}

function Show-ClientStatus {
    param([string]$Name, [string]$Path)
    if (Test-Path $Path) {
        Write-Host "  [x] $Name"
    } else {
        Write-Host "  [ ] $Name"
    }
}

function Install-ClientSkill {
    param([string]$Source, [string]$Target, [string]$Label)
    $base = "https://raw.githubusercontent.com/$Repo/main/skills"
    $commonDir = Join-Path $Target "common"
    if (-not (Test-Path $commonDir)) {
        New-Item -ItemType Directory -Path $commonDir | Out-Null
    }
    Write-Host "Installing $Label skill to $Target..."
    try {
        Invoke-WebRequest -Uri "$base/$Source/cc-session/SKILL.md" -OutFile (Join-Path $Target "SKILL.md") -UseBasicParsing
        Invoke-WebRequest -Uri "$base/common/resume-session.md" -OutFile (Join-Path $commonDir "resume-session.md") -UseBasicParsing
        Invoke-WebRequest -Uri "$base/common/close-session.md" -OutFile (Join-Path $commonDir "close-session.md") -UseBasicParsing
        Invoke-WebRequest -Uri "$base/common/review-history.md" -OutFile (Join-Path $commonDir "review-history.md") -UseBasicParsing
        Write-Host "$Label integration installed."
    }
    catch {
        Write-Error "Failed to download skill: $_"
        exit 1
    }
}

# ── getting started ───────────────────────────────────────────────────────────

function Show-NextSteps {
    Write-Host ""
    Write-Host "── Getting started ────────────────────────────────────────────────"
    Write-Host "  cc-session list          # 列出最近的 session"
    Write-Host "  cc-session read <id>     # 讀取對話內容"
    Write-Host "  /cc-session              # 在 Claude Code 中使用 (需已安裝 Skill)"
    Write-Host ""
    Write-Host "── Token counting (optional) ──────────────────────────────────────"
    Write-Host "  For precise token counts in 'cc-session stats', create:"
    Write-Host "  $SkillDir\config.json"
    Write-Host ""
    Write-Host '  {"anthropic_api_key_file": "<path-to-your-api-key-file>"}'
    Write-Host ""
}

# ── main ──────────────────────────────────────────────────────────────────────

$version = Get-LatestVersion
$arch    = Get-Architecture

Install-Binary -Version $version -Arch $arch
Update-UserPath
Install-Skill
Show-NextSteps
