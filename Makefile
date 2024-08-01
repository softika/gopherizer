# Simple Makefile for a Go project

# project name
PROJECT_NAME = TLDW

build:
	@echo "=== Building $(PROJECT_NAME)..."
	
	
	@go build -o $(PROJECT_NAME) main.go

# Run the application
run:
	@echo "=== Running server..."
	@[ -f ./config/config ] || { cp ./config/default.config ./config/config; }
	@go run main.go serve


# Create DB container
docker-run:
	@echo "=== Running docker containers..."
	@if docker compose up 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up; \
	fi

# Shutdown DB container
docker-down:
	@echo "=== Stopping docker containers..."
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi


# Test the application
test:
	@echo "=== Testing..."
	@go test -v ./...

race: # check race conditions
	@go test -v ./... --race

# Clean the binary
clean:
	@echo "=== Cleaning..."
	@rm -f $(PROJECT_NAME)

# Live Reload
watch:
	@if command -v air > /dev/null; then \
	    air; \
	    echo "Watching...";\
	else \
	    read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
	    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
	        go install github.com/air-verse/air@latest; \
	        air; \
	        echo "Watching...";\
	    else \
	        echo "You chose not to install air. Exiting..."; \
	        exit 1; \
	    fi; \
	fi

.PHONY: build run test clean
