.PHONY: build install test fmt vet clean tidy

BIN_DIR := bin
INSTALL_DIR := $(HOME)/bin
GOFLAGS := -trimpath
LDFLAGS := -s -w

build: $(BIN_DIR)/claude $(BIN_DIR)/claude-github-app

$(BIN_DIR)/claude:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/claude ./cmd/claude

$(BIN_DIR)/claude-github-app:
	@mkdir -p $(BIN_DIR)
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/claude-github-app ./cmd/claude-github-app

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
