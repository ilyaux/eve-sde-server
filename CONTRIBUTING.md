# Contributing

Thanks for helping improve EVE SDE Server. This project is intended to stay
easy to run locally, safe to operate, and useful for people building tools
around EVE Online static data.

## Before You Start

- Check existing issues before opening a new one.
- Keep changes focused. A small, well-tested pull request is easier to review
  and merge than a broad rewrite.
- For behavior changes, update README, OpenAPI, SDK docs, or examples when they
  are affected.
- Do not commit generated databases, SDE zip files, logs, binaries, or local
  `.env` files.

## Development Setup

```bash
git clone https://github.com/ilyaux/eve-sde-server.git
cd eve-sde-server

go mod download
make migrate
go run ./cmd/server
```

The sample migration creates a small SQLite database with enough data to test
the API. Use `make import-sde` when you need the full CCP SDE dataset.

## Quality Gates

Run these before opening a pull request:

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...

cd sdk/go
go vet ./...
go test ./...

cd examples/basic
go vet ./...
go test ./...
```

Also build the container when touching Docker files:

```bash
docker build -t eve-sde-server:local .
```

## Pull Requests

Good pull requests include:

- A clear summary of the behavior change.
- Tests or a reason tests are not needed.
- Notes about migrations, configuration changes, or compatibility.
- Screenshots or browser notes for admin UI changes.

## API Compatibility

Treat public REST paths, GraphQL fields, SDK method names, and response shapes as
compatibility surfaces. Prefer additive changes. If a breaking change is
unavoidable, call it out clearly in the pull request and update docs.

## Security

Do not open public issues for vulnerabilities. Follow [SECURITY.md](SECURITY.md).
