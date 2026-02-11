# EVE SDE Server

> **EVE Online API** | **Static Data Export** | **REST & GraphQL** | **Game Development** | **MMO Database**

Modern REST & GraphQL API for EVE Online Static Data Export (SDE) with auto-updates, full-text search, and production-ready features.

[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/docker-ready-blue.svg)](https://hub.docker.com)
[![EVE Online](https://img.shields.io/badge/EVE-Online-yellow.svg)](https://www.eveonline.com/)
[![API](https://img.shields.io/badge/API-REST%20%26%20GraphQL-blue.svg)](https://github.com/ilyaux/eve-sde-server)

---

## 🎯 What is This?

**EVE SDE Server** is a production-ready **Go microservice** that provides fast, searchable access to **EVE Online's Static Data Export (SDE)** through modern **REST and GraphQL APIs**.

Built for **EVE Online third-party developers**, this server replaces outdated Fuzzwork MySQL dumps with a **self-hosted, auto-updating solution** that synchronizes with CCP's official SDE daily. Perfect for building market analysis tools, fitting calculators, industry planners, Discord bots, mobile apps, and any EVE Online third-party application requiring item database access.

**Why use this?**
- ⚡ **10x faster** than parsing YAML files manually
- 🔄 **Always up-to-date** with automatic SDE synchronization
- 🚀 **Production-ready** with monitoring, caching, and security
- 🆓 **Free & Open Source** - host it yourself
- 📊 **Flexible querying** via REST or GraphQL

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

## 🎯 Use Cases & Examples

**For EVE Online Developers:**
- 🤖 **Discord Bots** - Item lookup commands, price checks, fitting links
- 📱 **Mobile Apps** - iOS/Android EVE companion apps with offline item database
- 🌐 **Web Tools** - Market analysis, industry calculators, fitting tools
- 📊 **Data Analytics** - Market trends, industry planning, mining optimization
- 🎮 **Third-Party Apps** - Ship fittings, skill planners, corporation tools

**Real-World Applications:**
- EVE Market Trading Platforms
- Industry & Manufacturing Calculators
- PvP Fitting Databases
- Mining Yield Calculators
- Corporation Asset Management
- Alliance Doctrine Builders
- Newbie Helper Bots
- EVE Wiki & Documentation Sites

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

## 🔍 Related Topics & Keywords

<details>
<summary>SEO Keywords for Discovery</summary>

**EVE Online Development:**
eve online api, eve online sde, eve online static data export, eve online database, eve online items database, eve online third party tools, eve online developer tools, eve online api wrapper, eve online sdk, eve sde server, eve online data access, eve online item search, eve online market tools

**API & Technology:**
rest api golang, graphql api go, sqlite fts5, full text search api, game database api, mmo database api, real-time game data, auto-updating api, docker microservice, prometheus monitoring, grafana dashboard, go chi router, golang rest server

**Game Development:**
game item database, mmo item database, spaceship game api, sci-fi game database, multiplayer game tools, game data synchronization, game api development, third party game tools, game developer api, indie game backend

**Use Cases:**
eve market analysis, eve trading tools, eve fitting tools, eve wiki api, eve discord bot, eve mobile app, eve third party app, eve online calculator, eve industry tools, eve manufacturing tools, eve mining tools, eve pvp tools

**Technologies:**
golang microservice, sqlite embedded database, docker compose deployment, kubernetes ready, prometheus metrics, grafana monitoring, graphql playground, rest pagination, api rate limiting, jwt authentication, tls https server, automated data updates

**Alternatives To:**
fuzzwork mysql dump, eve central api, zkillboard api, eve marketdata, eve online esi alternative, static data alternative, sde yaml parser, eve database hosting

</details>

---

<div align="center">
Made with ❤️ for the EVE Online developer community<br>
Not affiliated with CCP Games
</div>
