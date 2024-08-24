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
	@if docker compose up -d 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up -d; \
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


# migrate DB
migrate-up:
	@echo "=== Migrating database..."
	@[ -f ./config/config ] || { cp ./config/default.config ./config/config; }
	@go run main.go migrate up

# rollback DB
migrate-down:
	@echo "=== Rolling back database..."
	@[ -f ./config/config ] || { cp ./config/default.config ./config/config; }
	@go run main.go migrate down

# Test the application
test:
	@echo "=== Running tests with race detector"
	go test -vet=off -count=1 -race -timeout=30s ./...

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

mock:
	@go generate -x ./...

.PHONY: build run test clean mock watch
