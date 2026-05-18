# wsl-obsidian-clip

If you're on Windows and use WSL for your dev environment, and then want to use Obsidian for note taking - you will run into a bunch of annoying issues because Obsidian have a hard time translating the Windows/Linux path differences.

Thanks to WSLg we can install Obsidian as a Linux GUI app inside WSL and run it in Windows. Now we have the vault, terminal and tooling available in the same environment - nice. However, screenshots captured in Windows and stored only in the Windows clipboard cannot be pasted into Linux Obsidian as file attachments. Text clipboard sharing works, but image clipboard data does not arrive in a form that Obsidian can reliably treat as an attachment.

While looking for a solution I found [wsl-screenshot-cli](https://github.com/Nailuu/wsl-screenshot-cli) which is a very useful reference because it bridges the Windows clipboard from WSL and turns screenshots into saved files. Its main goal, though, is terminal-oriented paste behavior and Windows clipboard enrichment. That does not directly solve the Windows/WSL/Obsidian case, where the important outcome is to create an image file in the vault and make it easy to insert as an Obsidian embed.

## Problem

Screenshots captured in Windows (Win+Shift+S, Snipping Tool) can't be pasted as image attachments into Linux Obsidian running under WSLg. Text clipboard sharing works, but image data doesn't arrive in a form Obsidian can use.

## Solution

`wsl-obsidian-clip` is a polling daemon that:

1. Reads screenshot image data from the Windows clipboard via a persistent `powershell.exe -STA` subprocess
2. Offers an Obsidian embed (`![[2026-05-13_14-48-21.png]]`) on the X11 clipboard
3. Saves the PNG into your Obsidian vault's configured attachment directory

You then paste the embed text directly into your note, while the file automatically gets saved as an attachment with proper linking.

## Install

```bash
# Build
make build

# Optional: install to ~/.local/bin
make install
```

`make install` also creates `~/.config/wsl-obsidian-clip/config.toml` with commented examples if it does not already exist. If `~/.local/bin` is not already on PATH, the installer discovers your shell profile and adds an idempotent PATH snippet for future shells. The config file is optional; the default installation uses automatic discovery.

Requires:

- WSL2 with [interop](https://learn.microsoft.com/en-us/windows/wsl/wsl-config#interop-settings) enabled
- A WSLg graphical session with X11 clipboard access (`DISPLAY` must be set)
- Windows PowerShell available through WSL interop; the tool discovers it from PATH or the standard Windows location
- Go 1.23+ (the module targets Go 1.23)

## Usage

```bash
# Run in foreground
wsl-obsidian-clip start

# Run as background daemon
wsl-obsidian-clip start --daemon

# Check daemon status
wsl-obsidian-clip status

# Stop daemon
wsl-obsidian-clip stop
```

### Flags

| Flag         | Short | Default | Description                       |
| ------------ | ----- | ------- | --------------------------------- |
| `--daemon`   | `-d`  | `false` | Run as background daemon          |
| `--interval` | `-i`  | `250`   | Polling interval in ms (100–5000) |
| `--verbose`  | `-v`  | `false` | Log PowerShell I/O                |
| `--quiet`    | `-q`  | `false` | Suppress info messages            |

### Start from `.bashrc`

If you want the daemon to start automatically with interactive WSL shells, guard it so it only runs when a WSLg display is available:

```bash
if [[ $- == *i* && -n ${DISPLAY:-} ]]; then
  wsl-obsidian-clip start --daemon --quiet
fi
```

With `--quiet`, this command is idempotent: if the daemon is already running, it exits successfully without printing anything.

## How it works

```
┌────────────────────┐  stdin/stdout  ┌──────────────────┐
│  wsl-obsidian-clip │◄──────────────►│ powershell.exe   │
│  (Go daemon)       │   CHECK/IMAGE  │ -STA (clipboard) │
└──────┬─────────────┘                └──────────────────┘
       │
       ├─ Detects new screenshot image in Windows clipboard
       ├─ SHA256 dedup (skips if identical image already saved)
       ├─ Offers ![[filename.png]] on the X11 clipboard
       ├─ Saves PNG with timestamp filename on first paste
       └─ You paste the embed into your Obsidian note
```

## Vault detection

The tool auto-detects your active Obsidian vault by reading Obsidian's `obsidian.json` (prefers currently open vaults, then most recently used). It checks the standard XDG config path plus common Flatpak and Snap paths, then reads `.obsidian/app.json` to find the configured attachment folder path.

## Config overrides

If automatic discovery does not work on your distro or install method, set only the values you need in `~/.config/wsl-obsidian-clip/config.toml`:

```toml
# wsl-obsidian-clip configuration
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

# Override the PowerShell executable path when automatic discovery fails.
# powershell_path_override = "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"
```

Missing or commented values keep the default auto-discovery behavior.

## Acknowledgements

This project is very much inspired and based on [wsl-screenshot-cli](https://github.com/Nailuu/wsl-screenshot-cli) built by [Nailuu](https://github.com/Nailuu)
