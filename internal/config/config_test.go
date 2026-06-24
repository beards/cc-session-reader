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
