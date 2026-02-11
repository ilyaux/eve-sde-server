package config

import "github.com/caarlos0/env/v10"

type Config struct {
	Port           int    `env:"PORT" envDefault:"8080"`
	DBPath         string `env:"DB_PATH" envDefault:"data/sde.db"`
	TLSEnabled     bool   `env:"TLS_ENABLED" envDefault:"false"`
	TLSCertFile    string `env:"TLS_CERT_FILE" envDefault:""`
	TLSKeyFile     string `env:"TLS_KEY_FILE" envDefault:""`
	AllowedOrigins string `env:"ALLOWED_ORIGINS" envDefault:"*"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	return cfg, env.Parse(cfg)
}
