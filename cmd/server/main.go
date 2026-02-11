package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ilya/eve-sde-server/internal/api/handlers"
	apimiddleware "github.com/ilya/eve-sde-server/internal/api/middleware"
	"github.com/ilya/eve-sde-server/internal/auth"
	"github.com/ilya/eve-sde-server/internal/cache"
	"github.com/ilya/eve-sde-server/internal/config"
	"github.com/ilya/eve-sde-server/internal/database"
	"github.com/ilya/eve-sde-server/internal/scheduler"
)

func main() {
	// Setup logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Connect to database
	db, err := database.New(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}

	log.Info().Str("db_path", cfg.DBPath).Msg("database connected")

	// Setup auth manager
	authManager := auth.NewManager(db)

	// Setup cache (in-memory for now)
	cacheStore, err := cache.NewMemoryCache(60*time.Second, 100) // 60s TTL, 100MB max
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize cache")
	}
	log.Info().Msg("cache initialized (in-memory)")

	// Setup rate limiter
	rateLimiter := apimiddleware.NewRateLimiter()

	// Setup SDE auto-update scheduler
	sdeScheduler := scheduler.New(
		db,
		os.Getenv("SDE_URL"), // Empty = use default
		"data",               // Data directory
		strings.ToLower(os.Getenv("SDE_AUTO_UPDATE")) == "true",
	)
	if err := sdeScheduler.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start scheduler")
	}

	// Public endpoints (no API key auth required)
	// Note: /admin and /api/admin/* use HTTP Basic Auth instead
	publicEndpoints := map[string]bool{
		"/":                 true,
		"/health":           true,
		"/metrics":          true,
		"/docs":             true,
		"/api/openapi.yaml": true,
		// ESI proxy is public (no auth required)
	}

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(apimiddleware.Metrics) // Prometheus metrics
	// Configure CORS with allowed origins from config
	allowedOrigins := strings.Split(cfg.AllowedOrigins, ",")
	if cfg.AllowedOrigins == "*" {
		log.Warn().Msg("CORS configured with wildcard (*) - not recommended for production")
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Phase 3 middleware (optional, controlled by env vars)
	authEnabled := strings.ToLower(os.Getenv("AUTH_ENABLED")) == "true"
	if authEnabled {
		log.Info().Msg("authentication enabled")
		r.Use(apimiddleware.Auth(authManager, publicEndpoints))
		r.Use(apimiddleware.RateLimit(rateLimiter))
	} else {
		log.Warn().Msg("authentication disabled (set AUTH_ENABLED=true to enable)")
	}

	// Enable caching for API responses
	cacheMiddleware := apimiddleware.NewCacheMiddleware(cacheStore, 60*time.Second)
	log.Info().Msg("caching enabled (60s TTL)")

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"OK"}`))
	})

	// Metrics endpoint
	r.Handle("/metrics", promhttp.Handler())

	// Swagger UI - serve OpenAPI spec
	r.Get("/api/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "api/openapi.yaml")
	})

	// Swagger UI page
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/swagger.html")
	})

	// Admin Dashboard (protected with HTTP Basic Auth)
	r.With(apimiddleware.AdminAuth).Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/admin.html")
	})

	// Admin API routes (protected with HTTP Basic Auth)
	adminHandler := handlers.NewAdminHandler(db, authManager)
	schedulerHandler := handlers.NewSchedulerHandler(sdeScheduler)
	r.Route("/api/admin", func(r chi.Router) {
		r.Use(apimiddleware.AdminAuth) // Require admin auth for all /api/admin/* routes
		r.Get("/stats", adminHandler.Stats)
		r.Get("/keys", adminHandler.ListKeys)
		r.Post("/keys", adminHandler.CreateKey)
		r.Delete("/keys/{id}", adminHandler.RevokeKey)

		// SDE update management
		r.Post("/sde/update", schedulerHandler.TriggerUpdate)
		r.Get("/sde/status", schedulerHandler.GetStatus)
	})

	// API routes with caching
	itemHandler := handlers.NewItemHandler(db)
	diffHandler := handlers.NewDiffHandler(db)
	r.Route("/api/v1", func(r chi.Router) {
		// Apply cache middleware to all GET endpoints
		r.Use(cacheMiddleware.CacheResponse)

		r.Get("/items", itemHandler.List)
		r.Get("/items/{id}", itemHandler.Get)
		r.Get("/search", itemHandler.Search)

		// SDE diff and changelog
		r.Get("/diff", diffHandler.GetDiff)
		r.Get("/changelog", diffHandler.GetChangelog)
	})

	// GraphQL endpoint
	graphqlHandler, err := handlers.NewGraphQLHandler(db)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize GraphQL")
	}
	r.Handle("/api/graphql", graphqlHandler)
	log.Info().Msg("GraphQL endpoint available at /api/graphql (GraphiQL UI enabled)")

	// ESI Proxy routes (with caching)
	esiHandler := handlers.NewESIHandler()
	r.Route("/api/esi", func(r chi.Router) {
		// Specific endpoints with better caching
		r.Get("/types/{id}", esiHandler.GetTypeInfo)
		r.Get("/markets/prices", esiHandler.GetMarketPrices)
		r.Get("/markets/{regionID}/history/{typeID}", esiHandler.GetMarketHistory)

		// Generic proxy for any ESI endpoint
		r.HandleFunc("/*", esiHandler.Proxy)

		// Cache management
		r.Post("/cache/clear", esiHandler.ClearCache)
	})

	// Welcome page
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>EVE SDE Server</title>
    <style>
        body { font-family: system-ui; max-width: 800px; margin: 50px auto; padding: 20px; }
        code { background: #f4f4f4; padding: 2px 6px; border-radius: 3px; }
        pre { background: #f4f4f4; padding: 15px; border-radius: 5px; overflow-x: auto; }
    </style>
</head>
<body>
    <h1>🚀 EVE SDE Server</h1>
    <p>Modern REST API for EVE Online Static Data Export</p>

    <h2>Quick Examples</h2>

    <h3>Get item by ID</h3>
    <pre>curl http://localhost:%d/api/v1/items/34</pre>

    <h3>Search items</h3>
    <pre>curl "http://localhost:%d/api/v1/search?q=mineral"</pre>

    <h3>List all items</h3>
    <pre>curl http://localhost:%d/api/v1/items</pre>

    <h2>Available Endpoints</h2>
    <ul>
        <li><code>GET /health</code> - Health check</li>
        <li><code>GET /api/v1/items</code> - List items</li>
        <li><code>GET /api/v1/items/{id}</code> - Get item by ID</li>
        <li><code>GET /api/v1/search?q={query}</code> - Search items</li>
        <li><code>GET /docs</code> - Swagger UI documentation</li>
        <li><code>GET /admin</code> - Admin Dashboard (API key management)</li>
        <li><code>GET /metrics</code> - Prometheus metrics</li>
    </ul>

    <h2>Management</h2>
    <p>
        <a href="/admin" style="display: inline-block; background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold;">
            🔐 Admin Dashboard
        </a>
        <a href="/docs" style="display: inline-block; background: #48bb78; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; font-weight: bold; margin-left: 10px;">
            📚 API Documentation
        </a>
    </p>
</body>
</html>
`, cfg.Port, cfg.Port, cfg.Port)
	})

	// Start server with graceful shutdown
	addr := fmt.Sprintf(":%d", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("addr", addr).Bool("tls", cfg.TLSEnabled).Msg("🚀 server starting")

		if cfg.TLSEnabled {
			if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
				log.Fatal().Msg("TLS enabled but cert/key files not specified")
			}
			log.Info().
				Str("cert", cfg.TLSCertFile).
				Msgf("Starting HTTPS server on https://localhost:%d", cfg.Port)
			if err := srv.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("HTTPS server failed")
			}
		} else {
			log.Warn().Msg("Running without TLS - use only for development!")
			log.Info().Msgf("Open http://localhost:%d in your browser", cfg.Port)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatal().Err(err).Msg("HTTP server failed")
			}
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server gracefully...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop scheduler
	sdeScheduler.Stop()
	log.Info().Msg("Scheduler stopped")

	// Stop rate limiter cleanup goroutine
	rateLimiter.Stop()
	log.Info().Msg("Rate limiter stopped")

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped successfully")
}
