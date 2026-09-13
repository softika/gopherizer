# project name
PROJECT_NAME = gopherizer

## help: Show makefile commands.
.PHONY: help
help: Makefile
	@echo "===== Project: $(PROJECT_NAME) ====="
	@echo
	@echo " Usage: make <COMMAND>"
	@echo
	@echo " Available Commands:"
	@echo
	@sed -n 's/^##//p' $< | column -t -s ':' |  sed -e 's/^/ /'
	@echo

## start: Start all services from docker compose file.
.PHONY: start
start:
	@echo "=== Running docker compose..."
	@if docker compose up -d 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up -d; \
	fi

## stop: Stop all services from docker compose file.
.PHONY: stop
stop:
	@echo "=== Stopping docker compose..."
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

## observability: Start the full stack with Prometheus, Tempo and Grafana.
.PHONY: observability
observability:
	@echo "=== Starting observability stack..."
	@TRACING_ENABLED=true docker compose --profile observability up -d
	@echo
	@echo "    Grafana:     http://localhost:3000  <- view traces and dashboards here"
	@echo "    Prometheus:  http://localhost:9090"
	@echo "    Tempo:       localhost:3200          (API only, no web UI)"
	@echo

## observability-stop: Stop the observability stack.
.PHONY: observability-stop
observability-stop:
	@echo "=== Stopping observability stack..."
	@docker compose --profile observability down

## auth: Start the stack with Keycloak as the identity provider.
.PHONY: auth
auth:
	@echo "=== Starting stack with authentication..."
	@OIDC_ENABLED=true docker compose --profile auth up -d
	@echo
	@echo "    Keycloak:  http://localhost:8081  (admin / admin)"
	@echo "    Realm:     gopherizer"
	@echo
	@echo "    Get a token with every scope and the admin role:"
	@echo "      curl -s -d grant_type=password -d client_id=gopherizer-cli \\"
	@echo "        -d username=gopher -d password=gopher \\"
	@echo "        http://localhost:8081/realms/gopherizer/protocol/openid-connect/token | jq -r .access_token"
	@echo
	@echo "    The 'reader' user (reader / reader) holds no role, so it is"
	@echo "    refused by delete-profile and served by the rest."
	@echo

## auth-host: Start Keycloak for an application running on the host via 'make run'.
.PHONY: auth-host
auth-host:
	@echo "=== Starting Keycloak for a host-run application..."
	@KC_HOSTNAME=http://localhost:8081 docker compose --profile auth up -d keycloak database
	@echo
	@echo "    Keycloak issues tokens for http://localhost:8081, so export the"
	@echo "    matching issuer before 'make run':"
	@echo
	@echo "      export OIDC_ENABLED=true"
	@echo "      export OIDC_ISSUER=http://localhost:8081/realms/gopherizer"
	@echo "      export OIDC_AUDIENCE=gopherizer-api"
	@echo "      export OIDC_ROLES_CLAIM=realm_access.roles"
	@echo "      export OIDC_REQUIRE_TYPED_ACCESS_TOKEN=true"
	@echo

## auth-stop: Stop the identity provider.
.PHONY: auth-stop
auth-stop:
	@echo "=== Stopping the identity provider..."
	@docker compose --profile auth down

## build: Build the project.
.PHONY: build
build:
	@echo "=== Building $(PROJECT_NAME)..."
	@go build -o $(PROJECT_NAME) main.go

## run: Run the api server.
.PHONY: run
run:
	@echo "=== Running server..."
	@go run main.go serve

## db-start: Start database in a docker container.
.PHONY: db-start
db-start:
	@echo "=== Running database docker container..."
	@if docker compose up database -d 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose up -d; \
	fi

## db-stop: Shutdown database docker container.
.PHONY: db-stop
db-stop:
	@echo "=== Stopping database docker container..."
	@if docker compose down database 2>/dev/null; then \
		: ; \
	else \
		echo "Falling back to Docker Compose V1"; \
		docker-compose down; \
	fi

## migrate-up: Migrate the database.
.PHONY: migrate-up
migrate-up:
	@echo "=== Migrating database..."
	@go run main.go migrate up

## migrate-down: Rollback the database migration.
.PHONY: migrate-down
migrate-down:
	@echo "=== Rolling back database..."
	@go run main.go migrate down

## test: Run tests.
.PHONY: test
test:
	@echo "=== Running tests with race detector"
	go test -vet=off -count=1 -race -timeout=30s ./...

## clean: Clean the binary.
.PHONY: clean
clean:
	@echo "=== Cleaning..."
	@rm -f $(PROJECT_NAME)

## mocks: Generate mocks.
.PHONY: mocks
mocks:
	@go generate -x ./...
