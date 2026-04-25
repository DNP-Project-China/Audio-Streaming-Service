package repositories

import (
	"context"

	"github.com/DNP-Project-China/Audio-Streaming-Service/core-api/server"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/fx"
)

// DI for database
var Module = fx.Options(
	fx.Provide(
		fx.Annotate(
			NewPool,
			fx.As(new(DBTX)),
		),
		NewQueries,
	),
)

// DI constructor for PG connection pool
func NewPool(lc fx.Lifecycle, cfg *server.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL())

	if err != nil {
		return nil, err
	}

	// Ensure the pool is closed when the application stops
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			pool.Close()
			return nil
		},
	})

	return pool, nil
}

// DI constructor for Queries (database interactor)
func NewQueries(db DBTX) *Queries {
	return New(db)
}
