package repositories

import (
	"context"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewPool,
			fx.As(new(DBTX)),
		),
		NewQueries,
	),
)

// TODO: find a way to change context.Background() to app context
func NewPool(lc fx.Lifecycle, cfg *server.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL())

	if err != nil {
		return nil, err
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			pool.Close()
			return nil
		},
	})

	return pool, nil
}

func NewQueries(db DBTX) *Queries {
	return New(db)
}
