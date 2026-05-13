package obsidian

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Vault represents a detected Obsidian vault.
type Vault struct {
	Path           string
	AttachmentPath string // relative path from vault root, empty means vault root
}

// AttachmentDir returns the absolute path to the attachment directory.
func (v *Vault) AttachmentDir() string {
	if v.AttachmentPath == "" {
		return v.Path
	}
	return filepath.Join(v.Path, v.AttachmentPath)
}

// obsidianConfig represents the structure of ~/.config/obsidian/obsidian.json
type obsidianConfig struct {
	Vaults map[string]vaultEntry `json:"vaults"`
}

type vaultEntry struct {
	Path string `json:"path"`
	Ts   int64  `json:"ts"`
	Open bool   `json:"open"`
}

// appConfig represents relevant fields from .obsidian/app.json
type appConfig struct {
	AttachmentFolderPath string `json:"attachmentFolderPath"`
}

// DetectVault finds the most recently opened Obsidian vault and reads its config.
func DetectVault() (*Vault, error) {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "obsidian", "obsidian.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read obsidian config at %s: %w", configPath, err)
	}

	var config obsidianConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse obsidian config: %w", err)
	}

	if len(config.Vaults) == 0 {
		return nil, fmt.Errorf("no vaults found in obsidian config")
	}

	// Find vault: prefer open vaults, then most recently used
	type candidate struct {
		path string
		ts   int64
		open bool
	}
	var candidates []candidate
	for _, v := range config.Vaults {
		candidates = append(candidates, candidate{path: v.Path, ts: v.Ts, open: v.Open})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].open != candidates[j].open {
			return candidates[i].open
		}
		return candidates[i].ts > candidates[j].ts
	})

	vaultPath := candidates[0].path
	if _, err := os.Stat(vaultPath); err != nil {
		return nil, fmt.Errorf("vault path does not exist: %s", vaultPath)
	}

	vault := &Vault{Path: vaultPath}

	// Read attachment folder path from .obsidian/app.json
	appJsonPath := filepath.Join(vaultPath, ".obsidian", "app.json")
	appData, err := os.ReadFile(appJsonPath)
	if err == nil {
		var appCfg appConfig
		if err := json.Unmarshal(appData, &appCfg); err == nil && appCfg.AttachmentFolderPath != "" {
			vault.AttachmentPath = appCfg.AttachmentFolderPath
		}
	}
	// If app.json doesn't exist or has no attachment path, default to vault root

	return vault, nil
}
