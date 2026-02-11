# EVE SDE Server

Modern REST & GraphQL API for EVE Online Static Data Export (SDE) with auto-updates, full-text search, and production-ready features.

[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-blue.svg)](https://hub.docker.com)

---

## 🎯 What is This?

A production-ready Go server that provides fast, searchable access to EVE Online's Static Data Export through REST and GraphQL APIs. Replaces outdated MySQL dumps with a modern, auto-updating service.

**Key Features:**
- 🚀 **Fast** - SQLite with FTS5 full-text search (<50ms p95 latency)
- 🔄 **Auto-updating** - Daily SDE updates from CCP at 03:00 UTC
- 🔍 **Searchable** - Full-text search across all items
- 📊 **GraphQL** - Flexible queries with GraphiQL UI
- 🔒 **Secure** - TLS/HTTPS, API key auth, rate limiting
- 📦 **Single Binary** - No external dependencies (SQLite embedded)
- 🐳 **Docker Ready** - Includes Prometheus + Grafana monitoring

---

## 🚀 Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/ilyaux/eve-sde-server
cd eve-sde-server

# Start the server with monitoring
docker-compose up -d

# Server available at http://localhost:8080
# Grafana dashboard at http://localhost:3000 (admin/admin)
```

### Using Docker

```bash
docker run -p 8080:8080 -v sde-data:/app/data eve-sde-server
```

### Using Go (Development)

```bash
# Install dependencies
go mod download

# Run server
go run cmd/server/main.go

# Or build binary
make build
./bin/eve-sde-server
```

### Initial SDE Import

```bash
# Download and import SDE data (~400MB)
make import-sde

# Or manually
curl -L -o data/sde.zip https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/sde.zip
unzip data/sde.zip -d data/sde
go run cmd/import-sde/main.go
```

---

## 📖 API Documentation

### REST API

**Base URL:** `http://localhost:8080/api/v1`

#### Get Item by ID
```bash
GET /api/v1/items/{id}

# Example
curl http://localhost:8080/api/v1/items/34
```

**Response:**
```json
{
  "type_id": 34,
  "name": "Tritanium",
  "description": "A very heavy, yet bendable metal...",
  "volume": 0.01
}
```

#### Search Items
```bash
GET /api/v1/search?q={query}&limit={limit}&offset={offset}

# Example
curl "http://localhost:8080/api/v1/search?q=mineral&limit=5"
```

**Response:**
```json
{
  "data": [
    {
      "type_id": 34,
      "name": "Tritanium",
      "volume": 0.01
    }
  ],
  "meta": {
    "count": 1,
    "limit": 5,
    "offset": 0
  }
}
```

#### List Items
```bash
GET /api/v1/items?limit={limit}&offset={offset}

# Example
curl "http://localhost:8080/api/v1/items?limit=10"
```

---

### GraphQL API

**Endpoint:** `http://localhost:8080/api/graphql`

**GraphiQL UI:** Open `http://localhost:8080/api/graphql` in browser for interactive playground

#### Example Queries

**Get single item:**
```graphql
query {
  item(id: 34) {
    typeId
    name
    description
    volume
  }
}
```

**Search items:**
```graphql
query {
  search(query: "shield booster", limit: 5) {
    typeId
    name
    volume
  }
}
```

**List with pagination:**
```graphql
query {
  items(limit: 10, offset: 0) {
    typeId
    name
    volume
  }
}
```

---

## 🔧 Configuration

### Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
# Server
PORT=8080

# Database
DB_PATH=data/sde.db

# TLS/HTTPS
TLS_ENABLED=false
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem

# CORS
ALLOWED_ORIGINS=*

# Authentication
AUTH_ENABLED=false

# Auto-update
SDE_AUTO_UPDATE=true
SDE_URL=https://eve-static-data-export.s3-eu-west-1.amazonaws.com/tranquility/sde.zip
```

### Production Settings

For production deployment:

```bash
# Enable TLS
TLS_ENABLED=true
TLS_CERT_FILE=/etc/letsencrypt/live/yourdomain.com/fullchain.pem
TLS_KEY_FILE=/etc/letsencrypt/live/yourdomain.com/privkey.pem

# Set allowed origins (no wildcards!)
ALLOWED_ORIGINS=https://yourdomain.com,https://app.yourdomain.com

# Enable authentication
AUTH_ENABLED=true

# Enable auto-updates
SDE_AUTO_UPDATE=true
```

---

## 🔑 Authentication

When `AUTH_ENABLED=true`, API requests require an API key:

### Create API Key (Admin Dashboard)

1. Navigate to `http://localhost:8080/admin`
2. Login with default credentials (admin/admin) - **change in production!**
3. Create a new API key with custom rate limits

### Use API Key

```bash
# Include in Authorization header
curl -H "Authorization: Bearer esk_your_api_key_here" \
  http://localhost:8080/api/v1/items/34
```

### Admin API Endpoints

Protected with HTTP Basic Auth (admin/admin):

```bash
# Trigger manual SDE update
POST /api/admin/sde/update

# Get update status
GET /api/admin/sde/status

# Manage API keys
GET /api/admin/keys
POST /api/admin/keys
DELETE /api/admin/keys/{id}
```

---

## 📊 Monitoring

Included Prometheus + Grafana stack:

- **Prometheus:** `http://localhost:9090`
- **Grafana:** `http://localhost:3000` (admin/admin)
- **Metrics Endpoint:** `http://localhost:8080/metrics`

**Available Metrics:**
- Request count, duration, response codes
- Rate limiter stats
- Database connection pool
- Cache hit/miss rates

---

## 🛠️ Development

### Run Tests

```bash
# All tests
make test

# With coverage
make test-coverage

# Benchmarks
make bench
```

### Database Migrations

```bash
# Run migrations
make migrate

# Check migration status
make migrate-status

# Rollback last migration
make migrate-down
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Tidy modules
make mod-tidy
```

---

## 🐳 Docker

### Build Image

```bash
# Using make
make docker

# Or manually
docker build -t eve-sde-server .
```

### Run Container

```bash
# Basic run
docker run -p 8080:8080 -v sde-data:/app/data eve-sde-server

# With environment variables
docker run -p 8080:8080 \
  -e AUTH_ENABLED=true \
  -e SDE_AUTO_UPDATE=true \
  -v sde-data:/app/data \
  eve-sde-server
```

### Docker Compose

```bash
# Start all services
make docker-compose-up

# Stop all services
make docker-compose-down

# View logs
docker-compose logs -f eve-sde-server
```

---

## 📁 Project Structure

```
eve-sde-server/
├── cmd/
│   ├── server/          # Main server
│   ├── import-sde/      # SDE importer
│   └── migrate/         # Database migrations
├── internal/
│   ├── api/
│   │   ├── handlers/    # HTTP handlers
│   │   └── middleware/  # Auth, rate limiting, caching
│   ├── graphql/         # GraphQL schema & resolvers
│   ├── database/        # Database & migrations
│   ├── cache/           # In-memory cache (bigcache)
│   ├── auth/            # API key management
│   ├── scheduler/       # Auto-update scheduler
│   ├── esi/             # ESI proxy client
│   └── config/          # Configuration
├── web/                 # Admin dashboard HTML
├── docker-compose.yml   # Docker Compose config
├── Dockerfile           # Docker build
├── Makefile             # Build tasks
└── README.md
```

---

## ⚡ Performance

- **Latency (p95):** <30ms for item queries
- **Throughput:** >2000 req/s on modest hardware
- **Database:** SQLite with WAL mode + connection pooling
- **Caching:** 60s TTL in-memory cache (1024 shards)
- **Search:** FTS5 full-text index for instant search

---

## 🔒 Security Features

- ✅ **TLS/HTTPS** support with configurable certificates
- ✅ **API Key Authentication** with per-key rate limits
- ✅ **Rate Limiting** with token bucket algorithm
- ✅ **Input Validation** prevents SQL injection & DoS
- ✅ **CORS** configurable per environment
- ✅ **Graceful Shutdown** prevents data loss
- ✅ **SQL Injection Protection** via parameterized queries

---

## 🎯 Use Cases

- **EVE Online Developers** - Build apps with real-time SDE access
- **Third-Party Tools** - Integrate EVE item data via REST/GraphQL
- **Market Analysis** - Combine with ESI for market tools
- **Game Wikis** - Power search and item databases
- **Discord Bots** - Quick item lookups

---

## 📝 License

MIT License - see [LICENSE](LICENSE) for details

---

## 🤝 Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing`)
5. Open a Pull Request

---

## 📞 Support

- **Issues:** [GitHub Issues](https://github.com/ilyaux/eve-sde-server/issues)
- **Discussions:** [GitHub Discussions](https://github.com/ilyaux/eve-sde-server/discussions)

---

## 🙏 Credits

Built with:
- [Go](https://golang.org/) - Programming language
- [Chi](https://github.com/go-chi/chi) - HTTP router
- [SQLite](https://sqlite.org/) - Database
- [GraphQL-Go](https://github.com/graphql-go/graphql) - GraphQL implementation
- [Zerolog](https://github.com/rs/zerolog) - Structured logging
- [Prometheus](https://prometheus.io/) - Metrics & monitoring

Data provided by [CCP Games](https://www.ccpgames.com/) via EVE Online SDE.

---

<div align="center">
Made with ❤️ for the EVE Online developer community<br>
Not affiliated with CCP Games
</div>
