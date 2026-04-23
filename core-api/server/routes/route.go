package routes

import (
	"net/http"

	"go.uber.org/fx"
)

// Route interface for all HTTP handlers
type Route interface {
	http.Handler

	Pattern() string
	Method() string
}

// Helper functions for DI registration of routes and mux
func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Route)),
		fx.ResultTags(`group:"routes"`),
	)
}

// Helper function for DI registration of mux with all routes
func TakesRoutes(f any) any {
	return fx.Annotate(
		f,
		fx.ParamTags(`group:"routes"`),
	)
}
