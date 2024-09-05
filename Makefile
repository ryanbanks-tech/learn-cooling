# Define the directories
FRONTEND_DIR = frontend
BACKEND_DIR = backend

# Default target executed when no arguments are provided to 'make'
all: install build run

# Build the Go project
build:
	@echo "Building the project..."
	cd frontend && npm run build

# Install the dependencies
install:
	@echo "Installing the dependencies..."
	cd $(FRONTEND_DIR) && npm install

# Clean the build artifacts
clean:
	@echo "Cleaning up..."
	cd $(FRONTEND_DIR) && rm -rf build

# Run the Go project
run:
	@echo "Running the project..."
	cd $(BACKEND_DIR) && go run main.go

# Help command to display available targets
help:
	@echo "Makefile targets:"
	@echo "  make build   - Build the project"
	@echo "  make clean   - Remove build artifacts"
	@echo "  make run     - Run the project"
	@echo "  make all     - Build and run the project"
	@echo "  make help    - Display this help message"