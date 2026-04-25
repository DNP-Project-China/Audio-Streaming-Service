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

// Application DI entrypoint
func main() {
	fx.New(
		// Providing all the dependendencies
		fx.Provide(
			// Server configuration
			server.NewConfig,
			// Four routes
			routes.AsRoute(routes.NewHealthHandler),
			routes.AsRoute(handlers.NewUploadHandler),
			routes.AsRoute(handlers.NewTracksHandler),
			routes.AsRoute(handlers.NewDownloadHandler),
			// Web server itself
			routes.TakesRoutes(server.NewMux),
			server.NewHTTPServer,
		),
		// Database interactor
		repositories.Module,
		// S3 client
		storage.Module,
		// Business logic
		usecases.Module,
		// Kafka producer
		events.Module,

		// Start web server
		fx.Invoke(func(_ *http.Server) {

		}),
	).Run()
}
