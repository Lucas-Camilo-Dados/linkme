# Default recipe
default:
    @just --list

# Install dependencies
setup:
    go mod download

# Development server with live reload
dev:
    go run ./cmd/linkme watch

# One-shot build
build:
    go run ./cmd/linkme build

# Build and preview production output
preview: build
    cd dist && python3 -m http.server 3000

# Build Docker image locally
docker-build:
    docker build -t linkme:local .

# Run production image locally
docker-run: docker-build
    docker run --rm -p 8080:80 linkme:local

# Clean build artifacts
clean:
    rm -rf dist/
