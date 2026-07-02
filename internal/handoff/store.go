package handoff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var unsafePathChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func WriteArtifacts(storageRoot string, h Handoff, force bool) (string, error) {
	if storageRoot == "" {
		return "", fmt.Errorf("storage_root is required")
	}
	if h.Session.Provider == "" || h.Session.SessionID == "" {
		return "", fmt.Errorf("handoff session provider and session_id are required")
	}
	dir := filepath.Join(storageRoot, safeSegment(h.Session.Provider), safeSegment(h.Session.SessionID))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create handoff directory: %w", err)
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

func WriteFailedRaw(storageRoot, provider, sessionID, raw string) (string, error) {
	dir := filepath.Join(storageRoot, safeSegment(provider), safeSegment(sessionID))
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, "handoff.raw-failed.json")
	return path, atomicWrite(path, []byte(raw))
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
