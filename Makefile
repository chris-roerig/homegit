.PHONY: help build install clean uninstall release

BINARY_NAME=homegit
INSTALL_PATH=/usr/local/bin
CONFIG_DIR=$(HOME)/.homegit
HOMEBREW_TAP_DIR=../homebrew-homegit

.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build -o $(BINARY_NAME)

install: build ## Install homegit to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@if [ -f $(INSTALL_PATH)/$(BINARY_NAME) ]; then \
		echo "Stopping existing homegit server..."; \
		$(INSTALL_PATH)/$(BINARY_NAME) stop 2>/dev/null || true; \
	fi
	@sudo install -m 755 $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo ""
	@if [ ! -f $(CONFIG_DIR)/config ]; then \
		$(INSTALL_PATH)/$(BINARY_NAME) setup; \
	else \
		echo "Configuration already exists at $(CONFIG_DIR)/config"; \
		echo "Run 'homegit setup' to reconfigure or 'homegit config' to edit"; \
	fi
	@echo ""
	@echo "Installation complete!"

uninstall: ## Uninstall homegit
	@echo "Stopping homegit..."
	@$(INSTALL_PATH)/$(BINARY_NAME) stop 2>/dev/null || true
	@echo "Removing binary from $(INSTALL_PATH)..."
	@sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Remove config and repositories? (y/N): " && read ans && [ $${ans:-N} = y ] && \
		echo "Removing $(CONFIG_DIR)..." && rm -rf $(CONFIG_DIR) || \
		echo "Keeping $(CONFIG_DIR) (remove manually if needed)"
	@echo "Uninstall complete!"

clean: ## Remove built binary
	rm -f $(BINARY_NAME)

version: ## Bump version (usage: make version VERSION=v1.0.1)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make version VERSION=v1.0.1"; \
		exit 1; \
	fi
	@echo "Bumping version to $(VERSION)..."
	@VERSION_NUM=$$(echo $(VERSION) | sed 's/^v//'); \
	sed -i.bak "s/const Version = \".*\"/const Version = \"$$VERSION_NUM\"/" cmd/version.go && \
	rm -f cmd/version.go.bak
	@git add cmd/version.go
	@git diff --cached --quiet || (git commit -m "Bump version to $(VERSION)" && echo "Version bumped and committed!")
	@echo "Current version: $$VERSION_NUM"

release: ## Create a new release (usage: make release VERSION=v1.0.1)
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required. Usage: make release VERSION=v1.0.1"; \
		exit 1; \
	fi
	@echo "Creating release $(VERSION)..."
	@$(MAKE) version VERSION=$(VERSION)
	@echo "Pushing version bump..."
	@git push origin main
	@echo "Tagging release..."
	@git tag -d $(VERSION) 2>/dev/null || true
	@git push origin :refs/tags/$(VERSION) 2>/dev/null || true
	@git tag $(VERSION)
	@git push origin $(VERSION)
	@echo "Waiting for GitHub to process tag..."
	@sleep 5
	@echo "Calculating SHA256..."
	@SHA=$$(curl -sL https://github.com/chris-roerig/homegit/archive/refs/tags/$(VERSION).tar.gz | shasum -a 256 | cut -d' ' -f1); \
	echo "SHA256: $$SHA"; \
	if [ -d "$(HOMEBREW_TAP_DIR)" ]; then \
		echo "Updating Homebrew formula..."; \
		cd $(HOMEBREW_TAP_DIR) && \
		sed -i.bak "s|archive/refs/tags/v[0-9.]*\.tar\.gz|archive/refs/tags/$(VERSION).tar.gz|g" Formula/homegit.rb && \
		sed -i.bak "s/sha256 \"[a-f0-9]*\"/sha256 \"$$SHA\"/g" Formula/homegit.rb && \
		rm -f Formula/homegit.rb.bak && \
		git add Formula/homegit.rb && \
		git commit -m "Update homegit to $(VERSION)" && \
		git push && \
		echo "Homebrew tap updated!"; \
	else \
		echo "Warning: Homebrew tap directory not found at $(HOMEBREW_TAP_DIR)"; \
		echo "Please update Formula/homegit.rb manually with SHA256: $$SHA"; \
	fi
	@echo "Release $(VERSION) complete!"
