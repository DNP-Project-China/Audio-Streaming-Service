package server

import (
	"fmt"
	"net/url"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Port                  int           `env:"CORE_API_PORT" envDefault:"8000"`
	HTTPReadHeaderTimeout time.Duration `env:"CORE_API_READ_HEADER_TIMEOUT" envDefault:"5s"`
	HTTPReadTimeout       time.Duration `env:"CORE_API_READ_TIMEOUT" envDefault:"15s"`
	HTTPWriteTimeout      time.Duration `env:"CORE_API_WRITE_TIMEOUT" envDefault:"15s"`
	HTTPIdleTimeout       time.Duration `env:"CORE_API_IDLE_TIMEOUT" envDefault:"60s"`
	PostgresHost          string        `env:"POSTGRES_HOST" envDefault:"localhost"`
	PostgresPort          int           `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresUser          string        `env:"POSTGRES_USER,required"`
	PostgresPassword      string        `env:"POSTGRES_PASSWORD,required"`
	PostgresDB            string        `env:"POSTGRES_DB,required"`
	PostgresSSLMode       string        `env:"POSTGRES_SSLMODE" envDefault:"disable"`
}

func (c *Config) DatabaseURL() string {
	dbURL := &url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", c.PostgresHost, c.PostgresPort),
		Path:   c.PostgresDB,
		User:   url.UserPassword(c.PostgresUser, c.PostgresPassword),
	}

	query := url.Values{}
	query.Set("sslmode", c.PostgresSSLMode)
	dbURL.RawQuery = query.Encode()

	return dbURL.String()
}

func NewConfig() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
