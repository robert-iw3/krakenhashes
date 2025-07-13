# Get version from versions.json
VERSION := $(shell jq -r .backend ../versions.json)
LDFLAGS := -X github.com/ZerkerEOD/krakenhashes/backend/internal/version.Version=$(VERSION)

.PHONY: build
build:
	@echo "Building backend version $(VERSION)..."
	go build -ldflags="$(LDFLAGS)" -o ../bin/server/krakenhashes-server ./cmd/server

.PHONY: run
run: build
	@echo "Running backend version $(VERSION)..."
	../bin/server/krakenhashes-server

.PHONY: clean
clean:
	rm -rf ../bin/server/

.PHONY: test
test:
	go test -v ./...

.PHONY: test-auth
test-auth:
	@echo "Running authentication tests..."
	@JWT_SECRET=test-secret go test -v -count=1 ./pkg/jwt ./pkg/password ./internal/handlers/auth ./internal/integration_test

.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@JWT_SECRET=test-secret go test -v -count=1 ./internal/integration_test

.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	@go test -v -count=1 -short ./...

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: version
version:
	@echo "Backend version: $(VERSION)" 