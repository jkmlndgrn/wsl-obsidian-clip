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

Requires:

- WSL2 with [interop](https://learn.microsoft.com/en-us/windows/wsl/wsl-config#interop-settings) enabled
- `powershell.exe` accessible on PATH
- Go 1.22+ (build only)

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

The tool auto-detects your active Obsidian vault by reading `~/.config/obsidian/obsidian.json` (prefers currently open vaults, then most recently used). It reads `.obsidian/app.json` to find the configured attachment folder path.

## Acknowledgements

This project is very much inspired and based on [wsl-screenshot-cli](https://github.com/Nailuu/wsl-screenshot-cli) built by [Nailuu](https://github.com/Nailuu)
