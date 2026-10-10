APP_NAME ?= iotlab-service
MAIN_PATH ?= ./cmd/main.go
BINARY_DIR ?= build
BINARY_NAME ?= app-bin
DOCKER_IMAGE ?= iotlab-service:local

.PHONY: help run build test coverage e2e list-apis lint fmt check-comments check-swagger tidy docker-build clean mocks migrate-create swagger

help:
	@echo Available targets:
	@echo   make run              - Run the service locally
	@echo   make build            - Build binary into $(BINARY_DIR)/$(BINARY_NAME)
	@echo   make test             - Run all tests
	@echo   make coverage         - Gate A: run unit tests and enforce the 80% coverage gate
	@echo   make e2e              - Gate B: run the e2e suite and gate e2e API coverage
	@echo   make list-apis        - List every API with its e2e status
	@echo   make check-comments   - Check Go comments against go-style-core
	@echo   make check-swagger    - Check the OpenAPI spec against the actual routes
	@echo   make lint             - Run go vet
	@echo   make fmt              - Run go fmt on all packages
	@echo   make tidy             - Run go mod tidy
	@echo   make mocks            - Regenerate mocks via mockery (.mockery.yaml)
	@echo   make swagger          - Regenerate OpenAPI docs into docs/swagger (needs the swag CLI)
	@echo   make migrate-create name=xxx - Scaffold a new SQL migration pair (needs the migrate CLI)
	@echo   make docker-build     - Build Docker image $(DOCKER_IMAGE)
	@echo   make clean            - Remove build artifacts

run:
	go run $(MAIN_PATH)

build:
	@if not exist $(BINARY_DIR) mkdir $(BINARY_DIR)
	go build -o $(BINARY_DIR)/$(BINARY_NAME) $(MAIN_PATH)

test:
	go test ./...

# Gate A: fails when a test fails or unit coverage drops below 80%. CI does not run this — see
# .agents/skills/service-test/SKILL.md.
coverage:
	bash .agents/skills/service-test/scripts/coverage.sh --threshold 80

# Gate B: boots postgres/redis/emqx5, runs test/e2e, then requires an e2e test per selected API.
e2e:
	bash .agents/skills/service-test/scripts/e2e.sh

list-apis:
	bash .agents/skills/service-test/scripts/list-apis.sh

lint:
	go vet ./... && golangci-lint run ./...

# Comment rules from .agents/skills/go-style-core/SKILL.md; --all for the whole repo.
check-comments:
	bash .agents/skills/go-style-core/scripts/check-comments.sh --all

# Every business route must appear in docs/swagger/swagger.json; regenerate with `make swagger`.
check-swagger:
	bash .agents/skills/swagger-api/scripts/check-swagger.sh

fmt:
	go fmt ./...

tidy:
	go mod tidy

mocks:
	go tool mockery

# Regenerate the OpenAPI spec served at /swagger from the handler annotations.
# Requires the swag CLI: go install github.com/swaggo/swag/cmd/swag@v1.16.6
swagger:
	swag init -g $(MAIN_PATH) -o docs/swagger --parseInternal --parseDepth 3

# Scaffold a new migration pair under migrations/, e.g. `make migrate-create name=add_orders`.
# Requires the golang-migrate CLI (github.com/golang-migrate/migrate). The app runs migrations
# automatically at startup — this target is only for authoring new .sql files.
migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

dockerize:
	docker build -t $(DOCKER_IMAGE) .

clean:
	@if exist $(BINARY_DIR) rmdir /s /q $(BINARY_DIR)

