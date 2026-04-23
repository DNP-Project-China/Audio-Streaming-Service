package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server/routes"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

// HTTP server setup and lifecycle management
func withPermissiveCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Create a new Gorilla Mux router and register all routes
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

// Create and manage the HTTP server lifecycle with fx
func NewHTTPServer(lc fx.Lifecycle, cfg *Config, router *mux.Router) *http.Server {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           withPermissiveCORS(router),
		ReadHeaderTimeout: cfg.HTTPReadHeaderTimeout,
		ReadTimeout:       cfg.HTTPReadTimeout,
		WriteTimeout:      cfg.HTTPWriteTimeout,
		IdleTimeout:       cfg.HTTPIdleTimeout,
	}

	// Start the server in a separate goroutine
	// Shutdown gracefully when the application stops
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}

			logrus.WithField("port", srv.Addr).Info("Starting HTTP server")
			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logrus.WithError(err).Error("HTTP server stopped unexpectedly")
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})

	return srv
}
