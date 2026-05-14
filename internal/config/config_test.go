package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingConfigReturnsEmptyOverrides(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Config != (Config{}) {
		t.Fatalf("expected empty config, got %+v", loaded.Config)
	}
}

func TestLoadUsesXDGConfigHome(t *testing.T) {
	xdgConfigHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)
	t.Setenv("HOME", t.TempDir())

	configPath := filepath.Join(xdgConfigHome, "wsl-obsidian-clip", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("vault_path_override = \"/notes\"\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Path != configPath {
		t.Fatalf("Load path = %q, want %q", loaded.Path, configPath)
	}
	if loaded.Config.VaultPathOverride != "/notes" {
		t.Fatalf("VaultPathOverride = %q", loaded.Config.VaultPathOverride)
	}
}

func TestLoadInvalidTOMLReportsConfigPath(t *testing.T) {
	xdgConfigHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)

	configPath := filepath.Join(xdgConfigHome, "wsl-obsidian-clip", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("vault_path_override =\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), configPath) {
		t.Fatalf("expected parse error with config path, got %v", err)
	}
}
