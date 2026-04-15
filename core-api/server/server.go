package server

import (
	"fmt"
	"net/http"

	"go.uber.org/fx"
)

func NewHTTPServer(lc fx.Lifecycle, cfg *Config) *http.Server {
	srv := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Port)}
	return srv
}
