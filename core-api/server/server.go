package server

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server/routes"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

func NewMux(routes []routes.Route) *mux.Router {
	r := mux.NewRouter()

	for _, route := range routes {
		r.Handle(route.Pattern(), route).Methods(route.Method())
		logrus.WithFields(logrus.Fields{
			"method": route.Method(),
			"path":   route.Pattern(),
		}).Info("Registered route")
	}

	return r
}

func NewHTTPServer(lc fx.Lifecycle, cfg *Config, router *mux.Router) *http.Server {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}

			logrus.WithField("port", srv.Addr).Info("Starting HTTP server")
			go srv.Serve(ln)

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})

	return srv
}
