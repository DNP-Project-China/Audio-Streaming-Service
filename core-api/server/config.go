package server

import "github.com/caarlos0/env/v11"

type Config struct {
	Port int `env:"CORE_API_PORT" envDefault:"8000"`
}

func NewConfig() (*Config, error) {
	var cfg Config
	err := env.Parse(&cfg)

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
