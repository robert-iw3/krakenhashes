# Get version from versions.json
VERSION := $(shell jq -r .backend ../versions.json)
LDFLAGS := -X github.com/ZerkerEOD/krakenhashes/backend/internal/version.Version=$(VERSION)

.PHONY: build
build:
	@echo "Building backend version $(VERSION)..."
	go build -ldflags="$(LDFLAGS)" -o bin/krakenhashes-server ./cmd/server

.PHONY: run
run: build
	@echo "Running backend version $(VERSION)..."
	./bin/krakenhashes-server

.PHONY: clean
clean:
	rm -rf bin/

.PHONY: test
test:
	go test -v ./...

.PHONY: version
version:
	@echo "Backend version: $(VERSION)" 