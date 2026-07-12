package database

import (
	"clean-arch-template/config"
	"context"
	"fmt"
	"log/slog"
	"time"

	tx "github.com/Thiht/transactor/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	_defaultMaxPoolSize       = 4
	_defaultMinPoolSize       = 1
	_defaultConnAttempts      = 10
	_defaultConnTimeout       = 1
	_defaultHealthCheckPeriod = 1
)

// Postgres -.
type (
	Postgres struct {
		maxPoolSize       int32
		minPoolSize       int32
		connAttempts      int32
		connTimeout       int
		healthCheckPeriod int

		Pool       *pgxpool.Pool
		Transactor *tx.Transactor
		DBGetter   tx.DBGetter
	}
)

// New -.
func New(cfg *config.Config, opts ...Option) (*Postgres, error) {
	pg := &Postgres{
		maxPoolSize:       _defaultMaxPoolSize,
		minPoolSize:       _defaultMinPoolSize,
		connAttempts:      _defaultConnAttempts,
		connTimeout:       _defaultConnTimeout,
		healthCheckPeriod: _defaultHealthCheckPeriod,
	}

	// Custom options
	for _, opt := range opts {
		opt(pg)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("postgres - New - pgxpool.ParseConfig: %w", err)
	}

	// pgx pool settings
	setupPoolConfig(cfg, pg, poolConfig)

	for pg.connAttempts > 0 {
		pg.Pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err == nil {
			break
		}

		slog.Debug(fmt.Sprintf("Postgres is trying to connect to %s, attempts left: %d", cfg.DBHost, pg.connAttempts))

		time.Sleep(time.Duration(pg.connTimeout) * time.Second)

		pg.connAttempts--
	}

	if err != nil {
		return nil, fmt.Errorf("postgres - New - connAttempts == 0: %w", err)
	}

	// will use dbGetter in repositories
	// DBGetter is used to get the current DB handler from the context.
	// It returns the current transaction if there is one, otherwise it will return the original DB.
	pg.Transactor, pg.DBGetter = tx.NewTransactorFromPool(pg.Pool)

	return pg, nil
}

// Close -.
func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
