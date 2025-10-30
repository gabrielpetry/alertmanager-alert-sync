.PHONY: build run test clean docker-build docker-run fmt vet lint

# Binary name
BINARY_NAME=alertmanager-alert-sync

# Build the application
build:
	go build -o $(BINARY_NAME) ./cmd/alertmanager-alert-sync

# Run the application
run:
	go run ./cmd/alertmanager-alert-sync/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Clean build artifacts
clean:
	go clean
	rm -f $(BINARY_NAME)

# Build Docker image
docker-build:
	docker build -t $(BINARY_NAME):latest .

# Run Docker container
docker-run:
	docker run -p 8080:8080 \
		-e ALERTMANAGER_HOST=${ALERTMANAGER_HOST} \
		-e GRAFANA_IRM_URL=${GRAFANA_IRM_URL} \
		-e GRAFANA_IRM_TOKEN=${GRAFANA_IRM_TOKEN} \
		-e RECONCILE_INTERVAL=${RECONCILE_INTERVAL} \
		$(BINARY_NAME):latest

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run the application with hot reload (requires air)
dev:
	air

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  clean         - Clean build artifacts"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run linter"
	@echo "  deps          - Install dependencies"
	@echo "  dev           - Run with hot reload"
