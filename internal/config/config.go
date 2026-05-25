package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v10"
)

type Config struct {
	Port           int           `env:"PORT" envDefault:"8080"`
	DBPath         string        `env:"DB_PATH" envDefault:"data/sde.db"`
	TLSEnabled     bool          `env:"TLS_ENABLED" envDefault:"false"`
	TLSCertFile    string        `env:"TLS_CERT_FILE" envDefault:""`
	TLSKeyFile     string        `env:"TLS_KEY_FILE" envDefault:""`
	AllowedOrigins string        `env:"ALLOWED_ORIGINS" envDefault:"*"`
	AuthEnabled    bool          `env:"AUTH_ENABLED" envDefault:"false"`
	SDEAutoUpdate  bool          `env:"SDE_AUTO_UPDATE" envDefault:"false"`
	SDEURL         string        `env:"SDE_URL" envDefault:""`
	SDEDataDir     string        `env:"SDE_DATA_DIR" envDefault:"data"`
	CacheTTL       time.Duration `env:"CACHE_TTL" envDefault:"60s"`
	CacheMaxSizeMB int           `env:"CACHE_MAX_SIZE_MB" envDefault:"100"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("PORT must be between 1 and 65535")
	}
	if cfg.CacheTTL <= 0 {
		return nil, fmt.Errorf("CACHE_TTL must be greater than zero")
	}
	if cfg.CacheMaxSizeMB <= 0 {
		return nil, fmt.Errorf("CACHE_MAX_SIZE_MB must be greater than zero")
	}

	return cfg, nil
}
