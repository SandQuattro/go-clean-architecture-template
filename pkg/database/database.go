package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"clean-arch-template/config"

	"github.com/Masterminds/squirrel"
	tx "github.com/Thiht/transactor/pgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	_defaultMaxPoolSize  = 1
	_defaultConnAttempts = 10
	_defaultConnTimeout  = time.Second
	_defaultIsolation    = pgx.ReadCommitted
)

// Postgres -.
type (
	Postgres struct {
		maxPoolSize  int32
		connAttempts int32
		connTimeout  time.Duration
		isolation    pgx.TxIsoLevel

		Builder squirrel.StatementBuilderType

		Pool       *pgxpool.Pool
		Transactor *tx.Transactor
		DBGetter   tx.DBGetter
	}
)

// New -.
func New(cfg *config.Config, opts ...Option) (*Postgres, error) {
	databaseURL := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)

	pg := &Postgres{
		maxPoolSize:  _defaultMaxPoolSize,
		connAttempts: _defaultConnAttempts,
		connTimeout:  _defaultConnTimeout,
		isolation:    _defaultIsolation,
	}

	// Custom options
	for _, opt := range opts {
		opt(pg)
	}

	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres - NewPostgres - pgxpool.ParseConfig: %w", err)
	}

	poolConfig.MaxConns = pg.maxPoolSize

	for pg.connAttempts > 0 {
		pg.Pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err == nil {
			break
		}

		slog.Debug(fmt.Sprintf("Postgres is trying to connect, attempts left: %d", pg.connAttempts))

		time.Sleep(pg.connTimeout)

		pg.connAttempts--
	}

	if err != nil {
		return nil, fmt.Errorf("postgres - NewPostgres - connAttempts == 0: %w", err)
	}

	// adding squirrel statement builder, if you don't like raw sql
	pg.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

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
