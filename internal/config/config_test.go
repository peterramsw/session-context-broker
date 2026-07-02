package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeConfigFile(t *testing.T, dir string, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func writeRawConfigFile(t *testing.T, dir string, raw string) string {
	t.Helper()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestLoadFromPath_GivenValidConfigJSON_ThenPopulatesFields(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, map[string]any{
		"anthropic_api_key_file":   "/some/path/keys.env",
		"integration_test_session": "abc123",
		"no_usage":                 true,
	})

	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("CC_SESSION_NO_USAGE", "")

	cfg := LoadFromPath(path)

	if cfg.AnthropicAPIKeyFile != "/some/path/keys.env" {
		t.Errorf("AnthropicAPIKeyFile = %q, want /some/path/keys.env", cfg.AnthropicAPIKeyFile)
	}
	if cfg.IntegrationTestSession != "abc123" {
		t.Errorf("IntegrationTestSession = %q, want abc123", cfg.IntegrationTestSession)
	}
	if !cfg.NoUsage {
		t.Error("NoUsage = false, want true (from JSON)")
	}
}

func TestLoadFromPath_GivenTildeInKeyFilePath_ThenExpandsToHome(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, map[string]any{
		"anthropic_api_key_file": "~/.keys/anthropic.env",
	})
	t.Setenv("ANTHROPIC_API_KEY", "")

	cfg := LoadFromPath(path)

	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".keys/anthropic.env")
	if cfg.AnthropicAPIKeyFile != want {
		t.Errorf("AnthropicAPIKeyFile = %q, want %q", cfg.AnthropicAPIKeyFile, want)
	}
}

func TestLoadFromPath_GivenEnvVarAPIKey_ThenOverridesJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, map[string]any{})

	t.Setenv("ANTHROPIC_API_KEY", "env-key-value")

	cfg := LoadFromPath(path)

	if cfg.AnthropicAPIKey() != "env-key-value" {
		t.Errorf("AnthropicAPIKey() = %q, want env-key-value", cfg.AnthropicAPIKey())
	}
}

func TestLoadFromPath_GivenCCSessionNoUsageNonEmpty_ThenSetsNoUsage(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, map[string]any{"no_usage": false})

	t.Setenv("CC_SESSION_NO_USAGE", "1")

	cfg := LoadFromPath(path)

	if !cfg.NoUsage {
		t.Error("NoUsage = false, want true when CC_SESSION_NO_USAGE=1")
	}
}

// Guards the presence-based semantics: CC_SESSION_NO_USAGE="" must NOT enable NoUsage.
// A regression to Getenv (which conflates unset and empty) would make this fail.
func TestLoadFromPath_GivenCCSessionNoUsageEmpty_ThenDoesNotSetNoUsage(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, map[string]any{"no_usage": false})

	t.Setenv("CC_SESSION_NO_USAGE", "")

	cfg := LoadFromPath(path)

	if cfg.NoUsage {
		t.Error("NoUsage = true, want false when CC_SESSION_NO_USAGE is empty string")
	}
}

func TestFlexBool_GivenVariousTruthyValues_ThenAllParsedAsTrue(t *testing.T) {
	cases := []struct {
		label string
		json  string
	}{
		{"bool true", `{"no_usage": true}`},
		{"number 1", `{"no_usage": 1}`},
		{"string 1", `{"no_usage": "1"}`},
		{"string true", `{"no_usage": "true"}`},
		{"string yes", `{"no_usage": "yes"}`},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			dir := t.TempDir()
			path := writeRawConfigFile(t, dir, tc.json)
			t.Setenv("ANTHROPIC_API_KEY", "")
			t.Setenv("CC_SESSION_NO_USAGE", "")

			cfg := LoadFromPath(path)
			if !cfg.NoUsage {
				t.Errorf("NoUsage = false for JSON %s", tc.json)
			}
		})
	}
}

func TestFlexBool_GivenFalsyValues_ThenParsedAsFalse(t *testing.T) {
	cases := []struct {
		label string
		json  string
	}{
		{"bool false", `{"no_usage": false}`},
		{"number 0", `{"no_usage": 0}`},
		{"string 0", `{"no_usage": "0"}`},
		{"string no", `{"no_usage": "no"}`},
		{"string empty", `{"no_usage": ""}`},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			dir := t.TempDir()
			path := writeRawConfigFile(t, dir, tc.json)
			t.Setenv("ANTHROPIC_API_KEY", "")
			t.Setenv("CC_SESSION_NO_USAGE", "")

			cfg := LoadFromPath(path)
			if cfg.NoUsage {
				t.Errorf("NoUsage = true for JSON %s", tc.json)
			}
		})
	}
}

func TestLoadFromPath_GivenMissingConfigJSON_ThenReturnsZeroConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.json")
	t.Setenv("ANTHROPIC_API_KEY", "")

	cfg := LoadFromPath(path)

	if cfg.AnthropicAPIKeyFile != "" {
		t.Errorf("AnthropicAPIKeyFile = %q, want empty", cfg.AnthropicAPIKeyFile)
	}
	if cfg.NoUsage {
		t.Error("NoUsage = true, want false on missing config")
	}
	if cfg.AnthropicAPIKey() != "" {
		t.Errorf("AnthropicAPIKey() = %q, want empty", cfg.AnthropicAPIKey())
	}
}

func TestLoadSessionContextFromPath_GivenLocalLLMConfig_ThenLoadsNewSchema(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, map[string]any{
		"storage_root": "~/.session-context-test",
		"local_llm": map[string]any{
			"enabled":            true,
			"base_url":           "http://127.0.0.1:8080/v1",
			"api_key":            "",
			"model":              "local-model",
			"max_context":        1234,
			"max_output_tokens":  567,
			"timeout_seconds":    9,
			"min_filtered_chars": 321,
			"temperature":        0,
			"top_p":              0.95,
			"top_k":              20,
		},
	})

	cfg := LoadSessionContextFromPath(path)

	if !cfg.LocalLLM.IsEnabled() {
		t.Fatalf("LocalLLM.IsEnabled() = false, want true")
	}
	if cfg.LocalLLM.APIKey != "" {
		t.Fatalf("LocalLLM.APIKey = %q, want empty", cfg.LocalLLM.APIKey)
	}
	if cfg.LocalLLM.MaxContext != 1234 || cfg.LocalLLM.MaxOutputTokens != 567 || cfg.LocalLLM.TimeoutSeconds != 9 {
		t.Fatalf("LocalLLM numeric config = %#v", cfg.LocalLLM)
	}
	if cfg.LocalLLM.MinFilteredCharsOrDefault() != 321 {
		t.Fatalf("MinFilteredCharsOrDefault() = %d, want 321", cfg.LocalLLM.MinFilteredCharsOrDefault())
	}
	if cfg.LocalLLM.Temperature == nil || *cfg.LocalLLM.Temperature != 0 {
		t.Fatalf("Temperature = %#v, want pointer to 0", cfg.LocalLLM.Temperature)
	}
	if cfg.LocalLLM.TopP == nil || *cfg.LocalLLM.TopP != 0.95 || cfg.LocalLLM.TopK != 20 {
		t.Fatalf("sampling config = top_p:%#v top_k:%d", cfg.LocalLLM.TopP, cfg.LocalLLM.TopK)
	}
	if !filepath.IsAbs(cfg.StorageRoot) {
		t.Fatalf("StorageRoot was not expanded to absolute path: %q", cfg.StorageRoot)
	}
}

func TestLoadSessionContextFromPath_GivenLegacyQwenConfig_ThenMapsToLocalLLM(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, map[string]any{
		"qwen": map[string]any{
			"base_url": "http://127.0.0.1:8080/v1",
			"api_key":  "legacy-key",
			"model":    "local-model",
		},
	})

	cfg := LoadSessionContextFromPath(path)

	if !cfg.LocalLLM.IsEnabled() {
		t.Fatalf("legacy qwen config should enable LocalLLM: %#v", cfg.LocalLLM)
	}
	if cfg.LocalLLM.APIKey != "legacy-key" {
		t.Fatalf("LocalLLM.APIKey = %q, want legacy-key", cfg.LocalLLM.APIKey)
	}
}

func TestLoadSessionContextFromPath_GivenLocalLLMWithoutEnabled_ThenDisabled(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, map[string]any{
		"local_llm": map[string]any{
			"base_url": "http://127.0.0.1:8080/v1",
			"model":    "local-model",
		},
	})

	cfg := LoadSessionContextFromPath(path)

	if cfg.LocalLLM.IsEnabled() {
		t.Fatalf("LocalLLM.IsEnabled() = true, want false unless enabled is explicit")
	}
}

func TestLoadSessionContextFromPath_GivenEnvOverrides_ThenAppliesThem(t *testing.T) {
	dir := t.TempDir()
	path := writeConfigFile(t, dir, map[string]any{
		"local_llm": map[string]any{
			"enabled":  false,
			"base_url": "http://old/v1",
			"model":    "old",
		},
	})
	t.Setenv("SESSION_CONTEXT_LOCAL_LLM_ENABLED", "yes")
	t.Setenv("LOCAL_LLM_BASE_URL", "http://new/v1")
	t.Setenv("LOCAL_LLM_MODEL", "new-model")
	t.Setenv("LOCAL_LLM_MAX_CONTEXT", "4321")
	t.Setenv("LOCAL_LLM_MIN_FILTERED_CHARS", "8765")
	t.Setenv("LOCAL_LLM_TEMPERATURE", "0")
	t.Setenv("LOCAL_LLM_TOP_P", "0.9")
	t.Setenv("LOCAL_LLM_TOP_K", "10")

	cfg := LoadSessionContextFromPath(path)

	if !cfg.LocalLLM.IsEnabled() {
		t.Fatalf("LocalLLM.IsEnabled() = false, want env override enabled")
	}
	if cfg.LocalLLM.BaseURL != "http://new/v1" || cfg.LocalLLM.Model != "new-model" {
		t.Fatalf("env overrides not applied: %#v", cfg.LocalLLM)
	}
	if cfg.LocalLLM.MaxContext != 4321 {
		t.Fatalf("MaxContext = %d, want 4321", cfg.LocalLLM.MaxContext)
	}
	if cfg.LocalLLM.MinFilteredCharsOrDefault() != 8765 {
		t.Fatalf("MinFilteredCharsOrDefault() = %d, want 8765", cfg.LocalLLM.MinFilteredCharsOrDefault())
	}
	if cfg.LocalLLM.Temperature == nil || *cfg.LocalLLM.Temperature != 0 {
		t.Fatalf("Temperature = %#v, want 0", cfg.LocalLLM.Temperature)
	}
	if cfg.LocalLLM.TopP == nil || *cfg.LocalLLM.TopP != 0.9 || cfg.LocalLLM.TopK != 10 {
		t.Fatalf("sampling env overrides not applied: %#v", cfg.LocalLLM)
	}
}

func TestLocalLLMConfig_GivenNoThreshold_ThenUsesDefaultMinFilteredChars(t *testing.T) {
	cfg := LocalLLMConfig{}
	if cfg.MinFilteredCharsOrDefault() != DefaultMinFilteredChars {
		t.Fatalf("MinFilteredCharsOrDefault() = %d, want %d", cfg.MinFilteredCharsOrDefault(), DefaultMinFilteredChars)
	}
}

func TestGet_GivenReset_ThenReloadsConfig(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "first-key")
	t.Setenv("CC_SESSION_NO_USAGE", "1")
	Reset()
	cfg1 := Get()
	if cfg1.AnthropicAPIKey() != "first-key" {
		t.Fatalf("first Get() AnthropicAPIKey = %q, want first-key", cfg1.AnthropicAPIKey())
	}
	if !cfg1.NoUsage {
		t.Fatalf("first Get() NoUsage = false, want true")
	}

	t.Setenv("ANTHROPIC_API_KEY", "second-key")
	t.Setenv("CC_SESSION_NO_USAGE", "")
	Reset()
	cfg2 := Get()
	if cfg2.AnthropicAPIKey() != "second-key" {
		t.Errorf("after Reset() AnthropicAPIKey = %q, want second-key", cfg2.AnthropicAPIKey())
	}
	if cfg2.NoUsage {
		t.Errorf("after Reset() NoUsage = true, want false")
	}
}
