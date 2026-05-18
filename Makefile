.PHONY: build clean install

BINARY := wsl-obsidian-clip
INSTALL_DIR ?= $(HOME)/.local/bin
CONFIG_DIR := $(HOME)/.config/$(BINARY)
CONFIG_FILE := $(CONFIG_DIR)/config.toml

build:
	go build -o $(BINARY) .

install: build
	mkdir -p "$(INSTALL_DIR)"
	cp "$(BINARY)" "$(INSTALL_DIR)/$(BINARY)"
	mkdir -p "$(CONFIG_DIR)"
	@if [ ! -f "$(CONFIG_FILE)" ]; then \
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
		'# Override the PowerShell executable path when automatic discovery fails.' \
		'# powershell_path_override = "/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"' \
		> "$(CONFIG_FILE)"; \
	fi
	@sh scripts/ensure-path.sh "$(INSTALL_DIR)" "$(BINARY)"

clean:
	rm -f $(BINARY)
