BINARY = learn-cooling

# Default: build and run.
all: builds run

# Build the single self-contained binary (web assets are embedded via go:embed).
# Named "builds" to match the Render Build Command.
builds:
	@echo "Building $(BINARY)..."
	go build -o $(BINARY) .

# Run the compiled binary (this is the Render Start Command).
run:
	./$(BINARY)

# Run from source — handy during development.
dev:
	go run .

# Format and vet.
check:
	go fmt ./...
	go vet ./...

# Remove build artifacts.
clean:
	rm -f $(BINARY)

help:
	@echo "Makefile targets:"
	@echo "  make builds  - Build the binary (Render build command)"
	@echo "  make run     - Run the compiled binary (Render start command)"
	@echo "  make dev     - Run from source (go run)"
	@echo "  make check   - go fmt + go vet"
	@echo "  make clean   - Remove build artifacts"

.PHONY: all builds run dev check clean help
