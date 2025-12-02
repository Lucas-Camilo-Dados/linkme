# linkme

A customizable link page generator built in Go.

## Usage

```bash
# Build the static site
go run ./cmd/linkme build

# Serve locally
go run ./cmd/linkme serve
```

## Docker

```bash
docker run -d -p 8080:80 ghcr.io/ironicbadger/linkme:latest
```

## Configuration

Edit `config/config.yaml` to customize your links and appearance.
