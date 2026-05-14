package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const Template = `# wsl-obsidian-clip configuration
#
# All values are optional. Leave them commented out to use automatic discovery.

# Override the Obsidian vault path when auto-detection picks the wrong vault
# or cannot find your vault.
# vault_path_override = "/home/you/Notes"

# Override the attachment folder inside the vault.
# This is relative to the selected vault path.
# attachment_path_override = "Assets"

# Override the Obsidian config file path when your distro/package stores it
# somewhere the tool does not discover automatically.
# obsidian_config_path_override = "/home/you/.config/obsidian/obsidian.json"

# Override the PowerShell executable path when powershell.exe is not on PATH.
# powershell_path_override = "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"
`

type Config struct {
	VaultPathOverride          string `toml:"vault_path_override"`
	AttachmentPathOverride     string `toml:"attachment_path_override"`
	ObsidianConfigPathOverride string `toml:"obsidian_config_path_override"`
	PowerShellPathOverride     string `toml:"powershell_path_override"`
}

type Loaded struct {
	Config Config
	Path   string
}

func Load() (*Loaded, error) {
	path := Path()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Loaded{Path: path}, nil
		}
		return nil, fmt.Errorf("read config at %s: %w", path, err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config at %s: %w", path, err)
	}

	return &Loaded{Config: cfg, Path: path}, nil
}

func Path() string {
	if xdgConfigHome := os.Getenv("XDG_CONFIG_HOME"); xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "wsl-obsidian-clip", "config.toml")
	}
	return filepath.Join(homeDir(), ".config", "wsl-obsidian-clip", "config.toml")
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	return "."
}
