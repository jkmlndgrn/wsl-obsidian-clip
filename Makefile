.PHONY: build clean install

BINARY := wsl-obsidian-clip
CONFIG_DIR := $(HOME)/.config/$(BINARY)
CONFIG_FILE := $(CONFIG_DIR)/config.toml

build:
	go build -o $(BINARY) .

install: build
	mkdir -p $(HOME)/.local/bin
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)
	mkdir -p $(CONFIG_DIR)
	@if [ ! -f $(CONFIG_FILE) ]; then \
		printf '%s\n' \
		'# wsl-obsidian-clip configuration' \
		'#' \
		'# All values are optional. Leave them commented out to use automatic discovery.' \
		'' \
		'# Override the Obsidian vault path when auto-detection picks the wrong vault' \
		'# or cannot find your vault.' \
		'# vault_path_override = "/home/you/Notes"' \
		'' \
		'# Override the attachment folder inside the vault.' \
		'# This is relative to the selected vault path.' \
		'# attachment_path_override = "Assets"' \
		'' \
		'# Override the Obsidian config file path when your distro/package stores it' \
		'# somewhere the tool does not discover automatically.' \
		'# obsidian_config_path_override = "/home/you/.config/obsidian/obsidian.json"' \
		'' \
		'# Override the PowerShell executable path when powershell.exe is not on PATH.' \
		'# powershell_path_override = "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"' \
		> $(CONFIG_FILE); \
	fi

clean:
	rm -f $(BINARY)
