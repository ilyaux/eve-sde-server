# EVE SDE Server

[![CI](https://github.com/ilyaux/eve-sde-server/actions/workflows/ci.yml/badge.svg)](https://github.com/ilyaux/eve-sde-server/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/badge/go-1.25+-blue.svg)](https://go.dev/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Self-hosted REST and GraphQL API for the EVE Online Static Data Export (SDE).
It stores SDE data in SQLite, builds an FTS5 search index, and exposes a small
API that is easy to run locally, in Docker, or behind your own service.

## What It Does

- Imports CCP's official SDE zip into SQLite.
- Serves item lookup, taxonomy, list, and full-text search endpoints.
- Provides a GraphQL endpoint with GraphiQL enabled.
- Includes API key authentication, admin key management, and rate limiting.
- Stores API keys as SHA-256 hashes at rest; the raw key is only returned once.
- Proxies selected ESI endpoints with retry and in-memory caching.
- Ships with Prometheus metrics and Grafana provisioning.
- Includes a Go SDK under `sdk/go`.

## Requirements

- Go 1.25 or newer.
- Docker or Docker Compose if you want containerized deployment.
- About 400 MB of network download for a full SDE import.

## Quick Start With Go

```bash
git clone https://github.com/ilyaux/eve-sde-server.git
cd eve-sde-server

go mod download
make migrate
go run ./cmd/server
```

The server starts on `http://localhost:8080`.

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/items/34
curl "http://localhost:8080/api/v1/search?q=mineral&limit=5"
```

`make migrate` creates the SQLite schema and a small sample dataset. For real
data, import the full SDE:

```bash
make import-sde
```

## Docker

```bash
docker compose up --build
```

The container initializes the SQLite schema on startup. It does not download the
full SDE automatically; run `make import-sde` locally or execute
`./eve-sde-import-sde` inside a container that has the data volume mounted.

Services from `docker-compose.yml`:

- API server: `http://localhost:8080`
- Prometheus: `http://localhost:9090`
- Grafana: `http://localhost:3000` (`admin/admin`)

## Configuration

Environment variables:

| Variable | Default | Description |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port. |
| `DB_PATH` | `data/sde.db` | SQLite database path. |
| `TLS_ENABLED` | `false` | Enables HTTPS. |
| `TLS_CERT_FILE` | empty | TLS certificate path. |
| `TLS_KEY_FILE` | empty | TLS private key path. |
| `ALLOWED_ORIGINS` | `*` | Comma-separated CORS origins. |
| `AUTH_ENABLED` | `false` | Enables API key auth for API routes. |
| `ADMIN_USERNAME` | `admin` | Basic auth username for `/admin`. |
| `ADMIN_PASSWORD` | `admin` | Basic auth password for `/admin`. |
| `SDE_AUTO_UPDATE` | `false` | Enables scheduled SDE update checks. |
| `SDE_URL` | CCP SDE URL | SDE zip download URL. |

For public deployments, set explicit `ALLOWED_ORIGINS`, enable TLS at the edge,
and replace the default admin credentials.

## REST API

Base URL: `http://localhost:8080/api/v1`

```bash
GET /items?limit=50&offset=0
GET /items/{type_id}
GET /search?q=tritanium&limit=10&offset=0
GET /categories?limit=50&offset=0
GET /categories/{category_id}
GET /categories/{category_id}/groups
GET /groups?category_id=6&limit=50&offset=0
GET /groups/{group_id}
GET /diff?from=20250101&to=20250201
GET /changelog
```

List and search responses use this shape:

```json
{
  "data": [
    {
      "type_id": 34,
      "name": "Tritanium",
      "description": "A heavy, silver-gray metal...",
      "volume": 0.01,
      "group_id": 18,
      "category_id": 4
    }
  ],
  "meta": {
    "count": 1,
    "total": 1,
    "limit": 10,
    "offset": 0
  }
}
```

OpenAPI documentation is served at `http://localhost:8080/docs`.

## GraphQL

Endpoint: `http://localhost:8080/api/graphql`

```graphql
query {
  search(query: "shield booster", limit: 5) {
    typeId
    name
    volume
    groupId
    categoryId
  }
  groups(categoryId: 6, limit: 5) {
    groupId
    name
  }
}
```

## Authentication

When `AUTH_ENABLED=true`, REST and GraphQL API requests require an API key:

```bash
curl -H "Authorization: Bearer esk_your_api_key" \
  http://localhost:8080/api/v1/items/34
```

Admin routes use HTTP Basic Auth:

```bash
GET    /api/admin/stats
GET    /api/admin/keys
POST   /api/admin/keys
DELETE /api/admin/keys/{id}
POST   /api/admin/sde/update
GET    /api/admin/sde/status
```

The read-only ESI proxy is public. `POST /api/esi/cache/clear` is a mutating
maintenance endpoint and requires an API key when `AUTH_ENABLED=true`.

## Go SDK

```bash
go get github.com/ilyaux/eve-sde-server/sdk/go
```

```go
client := evesde.NewClient("http://localhost:8080", "")

item, err := client.GetItem(34)
results, err := client.Search("tritanium", 10)
list, err := client.ListItemsWithMeta(50, 0)
categories, err := client.ListCategories(50, 0)
groups, err := client.ListGroups(6, 50, 0)
```

## Development

```bash
go test ./...
go vet ./...
go build -o bin/eve-sde-server ./cmd/server
```

Useful make targets:

```bash
make migrate
make import-sde
make test
make docker-compose-up
make docker-compose-down
```

The CI workflow runs vet, unit tests, race tests on Linux, SDK checks, server
build, and Docker image build.

## Contributing and Security

Contributions are welcome. Start with [CONTRIBUTING.md](CONTRIBUTING.md) for
local setup, quality gates, and pull request expectations.

Please report vulnerabilities privately; see [SECURITY.md](SECURITY.md). Notable
changes are tracked in [CHANGELOG.md](CHANGELOG.md).

## Project Layout

```text
cmd/                       command binaries
internal/api/              HTTP handlers and middleware
internal/auth/             API key management
internal/cache/            memory and Redis cache implementations
internal/database/         SQLite setup and migrations
internal/graphql/          GraphQL schema and resolvers
internal/sde/              SDE downloader, parser, and importer
internal/scheduler/        scheduled SDE updates
sdk/go/                    Go SDK
api/openapi.yaml           OpenAPI spec
deployments/               Prometheus and Grafana provisioning
web/                       Swagger and admin HTML
```

## License

MIT. See [LICENSE](LICENSE).
