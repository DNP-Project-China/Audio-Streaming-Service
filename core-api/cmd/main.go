package main

import (
	"net/http"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server/routes"
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

		// Starting a web server
		fx.Invoke(func(srv *http.Server) {}),
	).Run()
}
