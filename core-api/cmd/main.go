package main

import (
	"net/http"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server/routes"
	"github.com/sirupsen/logrus"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			server.NewConfig,
			routes.AsRoute(routes.NewHealthHandler),
			routes.TakesRoutes(server.NewMux),
			server.NewHTTPServer,
		),
		fx.Invoke(func(cfg *server.Config, srv *http.Server) {
			logrus.WithField("port", srv.Addr).Info("Starting server")
		}),
	).Run()
}
