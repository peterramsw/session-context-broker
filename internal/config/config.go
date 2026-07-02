package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/Mapleeeeeeeeeee/cc-session-reader/internal/skillpath"
)

// flexBool unmarshals JSON booleans, numbers, and strings into a bool.
// Accepted truthy values: true, 1, "1", "true", "yes".
type flexBool bool

func (b *flexBool) UnmarshalJSON(data []byte) error {
	switch string(data) {
	case "true", "1", `"1"`, `"true"`, `"yes"`:
		*b = true
	default:
		*b = false
	}
	return nil
}

// Config holds resolved configuration from config.json and env var overrides.
type Config struct {
	AnthropicAPIKeyFile    string   `json:"anthropic_api_key_file"`
	IntegrationTestSession string   `json:"integration_test_session"`
	NoUsage                flexBool `json:"no_usage"`

	anthropicAPIKey string
}

func (c Config) AnthropicAPIKey() string { return c.anthropicAPIKey }

type SessionSourceConfig struct {
	Roots []string `json:"roots"`
}

const DefaultMinFilteredChars = 8000

type LocalLLMConfig struct {
	Enabled          flexBool `json:"enabled"`
	BaseURL          string   `json:"base_url"`
	APIKey           string   `json:"api_key"`
	Model            string   `json:"model"`
	MaxContext       int      `json:"max_context"`
	MaxOutputTokens  int      `json:"max_output_tokens"`
	TimeoutSeconds   int      `json:"timeout_seconds"`
	MinFilteredChars int      `json:"min_filtered_chars"`
	Temperature      *float64 `json:"temperature"`
	TopP             *float64 `json:"top_p"`
	TopK             int      `json:"top_k"`
}

func (c LocalLLMConfig) IsEnabled() bool {
	return bool(c.Enabled) && c.BaseURL != "" && c.Model != ""
}

func (c LocalLLMConfig) MinFilteredCharsOrDefault() int {
	if c.MinFilteredChars > 0 {
		return c.MinFilteredChars
	}
	return DefaultMinFilteredChars
}

type SessionContextConfig struct {
	SessionSources       map[string]SessionSourceConfig `json:"session_sources"`
	StorageRoot          string                         `json:"storage_root"`
	AllowedWorkspaceRoot []string                       `json:"allowed_workspace_roots"`
	LocalLLM             LocalLLMConfig                 `json:"local_llm"`

	legacyQwen LocalLLMConfig
}

var (
	once     sync.Once
	instance Config
)

// Get returns the singleton Config, loading it on first call.
func Get() Config {
	once.Do(func() {
		dir, err := skillpath.SkillDir()
		if err == nil {
			instance = LoadFromPath(filepath.Join(dir, "config.json"))
		}
	})
	return instance
}

// Reset clears the singleton so the next Get() reloads from disk.
// Intended for tests only.
func Reset() {
	once = sync.Once{}
}

func DefaultSessionContextConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".session-context", "config.json")
}

func LoadSessionContext() SessionContextConfig {
	if path := os.Getenv("SESSION_CONTEXT_CONFIG"); path != "" {
		return LoadSessionContextFromPath(path)
	}
	return LoadSessionContextFromPath(DefaultSessionContextConfigPath())
}

func LoadSessionContextFromPath(path string) SessionContextConfig {
	cfg := SessionContextConfig{
		SessionSources: map[string]SessionSourceConfig{},
		StorageRoot:    "~/.session-context",
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			var raw struct {
				SessionSources       map[string]SessionSourceConfig `json:"session_sources"`
				StorageRoot          string                         `json:"storage_root"`
				AllowedWorkspaceRoot []string                       `json:"allowed_workspace_roots"`
				LocalLLM             LocalLLMConfig                 `json:"local_llm"`
				Qwen                 LocalLLMConfig                 `json:"qwen"`
			}
			if json.Unmarshal(data, &raw) == nil {
				if raw.SessionSources != nil {
					cfg.SessionSources = raw.SessionSources
				}
				if raw.StorageRoot != "" {
					cfg.StorageRoot = raw.StorageRoot
				}
				cfg.AllowedWorkspaceRoot = raw.AllowedWorkspaceRoot
				cfg.LocalLLM = raw.LocalLLM
				cfg.legacyQwen = raw.Qwen
				if cfg.LocalLLM.BaseURL == "" && raw.Qwen.BaseURL != "" {
					cfg.LocalLLM = raw.Qwen
					cfg.LocalLLM.Enabled = true
				}
			}
		}
	}

	if cfg.StorageRoot == "" {
		cfg.StorageRoot = "~/.session-context"
	}
	cfg.StorageRoot = expandHome(cfg.StorageRoot)
	cfg.applySessionContextEnv()
	return cfg
}

func (c *SessionContextConfig) applySessionContextEnv() {
	if v := os.Getenv("SESSION_CONTEXT_STORAGE_ROOT"); v != "" {
		c.StorageRoot = expandHome(v)
	}
	if v := os.Getenv("SESSION_CONTEXT_LOCAL_LLM_ENABLED"); v != "" {
		c.LocalLLM.Enabled = parseFlexBool(v)
	} else if v := os.Getenv("LOCAL_LLM_ENABLED"); v != "" {
		c.LocalLLM.Enabled = parseFlexBool(v)
	}
	if v := os.Getenv("LOCAL_LLM_BASE_URL"); v != "" {
		c.LocalLLM.BaseURL = v
	}
	if v := os.Getenv("LOCAL_LLM_API_KEY"); v != "" {
		c.LocalLLM.APIKey = v
	}
	if v := os.Getenv("LOCAL_LLM_MODEL"); v != "" {
		c.LocalLLM.Model = v
	}
	if v := os.Getenv("LOCAL_LLM_MAX_CONTEXT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.LocalLLM.MaxContext = n
		}
	}
	if v := os.Getenv("LOCAL_LLM_MAX_OUTPUT_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.LocalLLM.MaxOutputTokens = n
		}
	}
	if v := os.Getenv("LOCAL_LLM_TIMEOUT_SECONDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.LocalLLM.TimeoutSeconds = n
		}
	}
	if v := os.Getenv("LOCAL_LLM_MIN_FILTERED_CHARS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.LocalLLM.MinFilteredChars = n
		}
	}
	if v := os.Getenv("LOCAL_LLM_TEMPERATURE"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			c.LocalLLM.Temperature = &n
		}
	}
	if v := os.Getenv("LOCAL_LLM_TOP_P"); v != "" {
		if n, err := strconv.ParseFloat(v, 64); err == nil {
			c.LocalLLM.TopP = &n
		}
	}
	if v := os.Getenv("LOCAL_LLM_TOP_K"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.LocalLLM.TopK = n
		}
	}
}

func parseFlexBool(value string) flexBool {
	switch value {
	case "true", "1", "yes", "TRUE", "YES", "on", "ON":
		return true
	default:
		return false
	}
}

func expandHome(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if len(path) == 1 {
		return home
	}
	if path[1] == '/' || path[1] == '\\' {
		return filepath.Join(home, path[2:])
	}
	return path
}

// LoadFromPath reads config.json from the given path and applies env var overrides.
// Missing or malformed config.json returns a zero Config.
func LoadFromPath(path string) Config {
	var cfg Config

	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	if len(cfg.AnthropicAPIKeyFile) > 0 && cfg.AnthropicAPIKeyFile[0] == '~' {
		home, err := os.UserHomeDir()
		if err == nil {
			cfg.AnthropicAPIKeyFile = filepath.Join(home, cfg.AnthropicAPIKeyFile[1:])
		}
	}

	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		cfg.anthropicAPIKey = key
	}

	if val, ok := os.LookupEnv("CC_SESSION_NO_USAGE"); ok && val != "" {
		cfg.NoUsage = true
	}

	return cfg
}
