# Define the directories
FRONTEND_DIR = frontend
BACKEND_DIR = backend

# Default target executed when no arguments are provided to 'make'
all: install react-build go-build run

# Build both the frontend and backend
builds: install react-build go-build

# Build the Go project
react-build:
	@echo "Building the project..."
	cd frontend && npm run build

# Build the Go project
go-build:
	@echo "Building the binary..."
	cd $(BACKEND_DIR) && go build -o learn-cooling

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
	cd $(BACKEND_DIR) && ./learn-cooling

# Help command to display available targets
help:
	@echo "Makefile targets:"
	@echo "  make all     		- Build and run the project"
	@echo "  make builds  		- Build the project and the go binary"
	@echo "  make react-build   - Build the project"
	@echo "  make go-build   	- Build the go binary"
	@echo "  make install  		- Install the dependencies"
	@echo "  make clean   		- Remove build artifacts"
	@echo "  make run     		- Run the project"
	@echo "  make help    		- Display this help message"