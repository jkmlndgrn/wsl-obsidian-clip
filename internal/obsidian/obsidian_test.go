package obsidian

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/config"
)

func TestDetectVaultUsesVaultAndAttachmentOverrides(t *testing.T) {
	vaultPath := t.TempDir()

	vault, err := DetectVault(config.Config{
		VaultPathOverride:      vaultPath,
		AttachmentPathOverride: "Assets",
	})
	if err != nil {
		t.Fatalf("DetectVault: %v", err)
	}
	if vault.Path != vaultPath {
		t.Fatalf("vault path = %q, want %q", vault.Path, vaultPath)
	}
	if vault.AttachmentPath != "Assets" {
		t.Fatalf("attachment path = %q", vault.AttachmentPath)
	}
	if got := vault.AttachmentDir(); got != filepath.Join(vaultPath, "Assets") {
		t.Fatalf("AttachmentDir = %q", got)
	}
}

func TestDetectVaultUsesObsidianConfigOverrideAndSkipsStaleVault(t *testing.T) {
	vaultPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vaultPath, ".obsidian"), 0755); err != nil {
		t.Fatalf("mkdir .obsidian: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vaultPath, ".obsidian", "app.json"), []byte(`{"attachmentFolderPath":"Attachments"}`), 0644); err != nil {
		t.Fatalf("write app.json: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "obsidian.json")
	data := fmt.Sprintf(`{"vaults":{"stale":{"path":"%s","ts":20,"open":true},"good":{"path":"%s","ts":10,"open":false}}}`, filepath.Join(t.TempDir(), "missing"), vaultPath)
	if err := os.WriteFile(configPath, []byte(data), 0644); err != nil {
		t.Fatalf("write obsidian config: %v", err)
	}

	vault, err := DetectVault(config.Config{ObsidianConfigPathOverride: configPath})
	if err != nil {
		t.Fatalf("DetectVault: %v", err)
	}
	if vault.Path != vaultPath {
		t.Fatalf("vault path = %q, want %q", vault.Path, vaultPath)
	}
	if vault.AttachmentPath != "Attachments" {
		t.Fatalf("attachment path = %q", vault.AttachmentPath)
	}
}

func TestDetectVaultAttachmentOverrideWinsOverAppJSON(t *testing.T) {
	vaultPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(vaultPath, ".obsidian"), 0755); err != nil {
		t.Fatalf("mkdir .obsidian: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vaultPath, ".obsidian", "app.json"), []byte(`{"attachmentFolderPath":"AppAssets"}`), 0644); err != nil {
		t.Fatalf("write app.json: %v", err)
	}

	configPath := filepath.Join(t.TempDir(), "obsidian.json")
	data := fmt.Sprintf(`{"vaults":{"good":{"path":"%s","ts":10,"open":true}}}`, vaultPath)
	if err := os.WriteFile(configPath, []byte(data), 0644); err != nil {
		t.Fatalf("write obsidian config: %v", err)
	}

	vault, err := DetectVault(config.Config{
		ObsidianConfigPathOverride: configPath,
		AttachmentPathOverride:     "OverrideAssets",
	})
	if err != nil {
		t.Fatalf("DetectVault: %v", err)
	}
	if vault.AttachmentPath != "OverrideAssets" {
		t.Fatalf("attachment path = %q", vault.AttachmentPath)
	}
}
