package main

import (
	"net/http"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/events"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/repositories"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server/handlers"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server/routes"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/storage"
	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/usecases"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		fx.Provide(
			server.NewConfig,
			routes.AsRoute(routes.NewHealthHandler),
			routes.AsRoute(handlers.NewUploadHandler),
			routes.TakesRoutes(server.NewMux),
			server.NewHTTPServer,
		),
		repositories.Module,
		storage.Module,
		usecases.Module,
		events.Module,

		// Start web server
		fx.Invoke(func(_ *http.Server) {

		}),
	).Run()
}
