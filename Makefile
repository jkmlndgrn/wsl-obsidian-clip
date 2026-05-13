.PHONY: build clean install

BINARY := wsl-obsidian-clip

build:
	go build -o $(BINARY) .

install: build
	cp $(BINARY) $(HOME)/.local/bin/$(BINARY)

clean:
	rm -f $(BINARY)
