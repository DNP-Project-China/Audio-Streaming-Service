package routes

import (
	"net/http"

	"go.uber.org/fx"
)

type Route interface {
	http.Handler

	Pattern() string
	Method() string
}

func AsRoute(f any) any {
	return fx.Annotate(
		f,
		fx.As(new(Route)),
		fx.ResultTags(`group:"routes"`),
	)
}

func TakesRoutes(f any) any {
	return fx.Annotate(
		f,
		fx.ParamTags(`group:"routes"`),
	)
}
