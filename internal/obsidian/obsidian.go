package obsidian

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jkmlndgrn/wsl-obsidian-clip/internal/config"
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

// DetectVault finds the selected Obsidian vault and reads its attachment config.
func DetectVault(cfg config.Config) (*Vault, error) {
	if cfg.VaultPathOverride != "" {
		return vaultFromPath(cfg.VaultPathOverride, cfg.AttachmentPathOverride)
	}

	configPath, data, err := readObsidianConfig(cfg.ObsidianConfigPathOverride)
	if err != nil {
		return nil, err
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

	for _, candidate := range candidates {
		vault, err := vaultFromPath(candidate.path, cfg.AttachmentPathOverride)
		if err == nil {
			return vault, nil
		}
	}

	return nil, fmt.Errorf("no usable vault paths found in obsidian config at %s", configPath)
}

func vaultFromPath(vaultPath string, attachmentOverride string) (*Vault, error) {
	if _, err := os.Stat(vaultPath); err != nil {
		return nil, fmt.Errorf("vault path does not exist: %s", vaultPath)
	}

	vault := &Vault{Path: vaultPath}
	if attachmentOverride != "" {
		vault.AttachmentPath = attachmentOverride
		return vault, nil
	}

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

func readObsidianConfig(overridePath string) (string, []byte, error) {
	for _, path := range obsidianConfigPaths(overridePath) {
		data, err := os.ReadFile(path)
		if err == nil {
			return path, data, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", nil, fmt.Errorf("read obsidian config at %s: %w", path, err)
		}
	}

	if overridePath != "" {
		return "", nil, fmt.Errorf("read obsidian config at %s: %w", overridePath, os.ErrNotExist)
	}
	return "", nil, fmt.Errorf("obsidian config not found; checked: %v", obsidianConfigPaths(""))
}

func obsidianConfigPaths(overridePath string) []string {
	if overridePath != "" {
		return []string{overridePath}
	}

	home := os.Getenv("HOME")
	paths := make([]string, 0, 4)
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		paths = append(paths, filepath.Join(xdgConfigHome, "obsidian", "obsidian.json"))
	}
	paths = append(paths,
		filepath.Join(home, ".config", "obsidian", "obsidian.json"),
		filepath.Join(home, ".var", "app", "md.obsidian.Obsidian", "config", "obsidian", "obsidian.json"),
		filepath.Join(home, "snap", "obsidian", "current", ".config", "obsidian", "obsidian.json"),
	)
	return paths
}
