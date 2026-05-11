.PHONY: build build-shims install install-shims install-all test fmt vet clean tidy

BIN_DIR := bin
INSTALL_DIR := $(HOME)/bin
GOFLAGS := -trimpath
LDFLAGS := -s -w

build: $(BIN_DIR)/claude $(BIN_DIR)/claude-github-app

build-shims: $(BIN_DIR)/gh $(BIN_DIR)/git

$(BIN_DIR)/claude:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/claude ./cmd/claude

$(BIN_DIR)/claude-github-app:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/claude-github-app ./cmd/claude-github-app

$(BIN_DIR)/gh:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gh ./cmd/gh

$(BIN_DIR)/git:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/git ./cmd/git

install: build
	@mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BIN_DIR)/claude $(BIN_DIR)/claude-github-app $(INSTALL_DIR)/
	@echo
	@echo "Installed:"
	@echo "  $(INSTALL_DIR)/claude"
	@echo "  $(INSTALL_DIR)/claude-github-app"
	@echo
	@echo "If $$HOME/bin is not first on PATH, add this to ~/.zshrc:"
	@echo '  export PATH="$$HOME/bin:$$PATH"'

# install-shims globally shadows gh and git on PATH. Every gh/git invocation
# (inside or outside a claude session) will route through the shim, which
# re-mints the GitHub App token if stale. Unmapped CWDs are passed through
# to the real binary unmodified, so it's safe — but it IS shadowing.
install-shims: build-shims
	@mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BIN_DIR)/gh $(BIN_DIR)/git $(INSTALL_DIR)/
	@echo
	@echo "Installed:"
	@echo "  $(INSTALL_DIR)/gh   (shadows real gh inside mapped CWDs)"
	@echo "  $(INSTALL_DIR)/git  (shadows real git inside mapped CWDs)"
	@echo
	@echo "Verify shadowing order with: which -a gh git"

install-all: install install-shims

test:
	go test ./... -race -count=1

fmt:
	gofmt -s -w .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BIN_DIR)
