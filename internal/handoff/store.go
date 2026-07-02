package handoff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var unsafePathChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func WriteArtifacts(storageRoot string, h Handoff, force bool) (string, error) {
	dir, err := ensureSessionDir(storageRoot, h.Session.Provider, h.Session.SessionID)
	if err != nil {
		return "", err
	}
	jsonPath := filepath.Join(dir, "handoff.json")
	mdPath := filepath.Join(dir, "handoff.md")
	if !force {
		if _, err := os.Stat(jsonPath); err == nil {
			return "", fmt.Errorf("handoff already exists at %s; rerun with --force to overwrite", jsonPath)
		}
	}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal handoff: %w", err)
	}
	if err := atomicWrite(jsonPath, append(data, '\n')); err != nil {
		return "", err
	}
	if err := atomicWrite(mdPath, []byte(RenderMarkdown(h))); err != nil {
		return "", err
	}
	return dir, nil
}

func WriteFilteredArtifact(storageRoot string, info SessionInfo, filtered string, force bool) (string, error) {
	dir, err := ensureSessionDir(storageRoot, info.Provider, info.SessionID)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "filtered.md")
	if !force {
		if _, err := os.Stat(path); err == nil {
			return "", fmt.Errorf("filtered artifact already exists at %s; rerun with --force to overwrite", path)
		}
	}
	if err := atomicWrite(path, []byte(renderFilteredArtifact(info, filtered))); err != nil {
		return "", err
	}
	return path, nil
}

func WriteFailedRaw(storageRoot, provider, sessionID, raw string) (string, error) {
	dir, err := ensureSessionDir(storageRoot, provider, sessionID)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "handoff.raw-failed.json")
	return path, atomicWrite(path, []byte(raw))
}

func ensureSessionDir(storageRoot, provider, sessionID string) (string, error) {
	if storageRoot == "" {
		return "", fmt.Errorf("storage_root is required")
	}
	if provider == "" || sessionID == "" {
		return "", fmt.Errorf("handoff session provider and session_id are required")
	}
	dir := filepath.Join(storageRoot, safeSegment(provider), safeSegment(sessionID))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create handoff directory: %w", err)
	}
	return dir, nil
}

func renderFilteredArtifact(info SessionInfo, filtered string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Filtered Session\n\n")
	fmt.Fprintf(&b, "Provider: %s\n\n", info.Provider)
	fmt.Fprintf(&b, "Session: %s\n\n", info.SessionID)
	if info.SourcePath != "" {
		fmt.Fprintf(&b, "Source: %s\n\n", info.SourcePath)
	}
	if info.Workspace != "" {
		fmt.Fprintf(&b, "Workspace: %s\n\n", info.Workspace)
	}
	fmt.Fprintf(&b, "Raw chars: %d\n\nFiltered chars: %d\n\n", info.RawChars, info.FilteredChars)
	b.WriteString(filtered)
	if !strings.HasSuffix(filtered, "\n") {
		b.WriteString("\n")
	}
	return b.String()
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename temp file: %w", err)
	}
	return nil
}

func safeSegment(value string) string {
	if value == "" {
		return "_"
	}
	return unsafePathChars.ReplaceAllString(value, "_")
}
