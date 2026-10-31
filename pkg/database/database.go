package database

import (
	"clean-arch-template/pkg/logger"
	"context"
	"fmt"
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
		debug             bool

		logger logger.Logger

		Pool       *pgxpool.Pool
		Transactor *tx.Transactor
		DBGetter   tx.DBGetter
	}
)

// New -.
func New(ctx context.Context, dsn string, opts ...Option) (*Postgres, error) {
	pg := &Postgres{
		maxPoolSize:       _defaultMaxPoolSize,
		minPoolSize:       _defaultMinPoolSize,
		connAttempts:      _defaultConnAttempts,
		connTimeout:       _defaultConnTimeout,
		healthCheckPeriod: _defaultHealthCheckPeriod,
		logger:            logger.Nop(),
	}

	// Custom options
	for _, opt := range opts {
		opt(pg)
	}

	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres - New - pgxpool.ParseConfig: %w", err)
	}

	// pgx pool settings
	//nolint:contextcheck // хуки пула без ctx в сигнатуре (AfterRelease, BeforeClose) логируют с context.Background()
	setupPoolConfig(pg, poolConfig)

	// NewWithConfig не подключается к БД: соединения создаются в фоне,
	// поэтому доступность проверяется явным Ping.
	pg.Pool, err = pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("postgres - New - pgxpool.NewWithConfig: %w", err)
	}

	if err := pg.waitForConnection(ctx, poolConfig.ConnConfig.Host); err != nil {
		pg.Pool.Close()
		return nil, err
	}

	// will use dbGetter in repositories
	// DBGetter is used to get the current DB handler from the context.
	// It returns the current transaction if there is one, otherwise it will return the original DB.
	pg.Transactor, pg.DBGetter = tx.NewTransactorFromPool(pg.Pool)

	return pg, nil
}

// waitForConnection делает до connAttempts попыток Ping с паузой connTimeout
// между ними; отмена ctx прерывает ожидание.
func (p *Postgres) waitForConnection(ctx context.Context, host string) error {
	for attempts := p.connAttempts; ; attempts-- {
		err := p.Pool.Ping(ctx)
		if err == nil {
			return nil
		}
		if attempts <= 1 {
			return fmt.Errorf("postgres - New - connAttempts == 0: %w", err)
		}

		p.logger.Debug(ctx, fmt.Sprintf("Postgres is trying to connect to %s, attempts left: %d", host, attempts-1))

		select {
		case <-ctx.Done():
			return fmt.Errorf("postgres - New - waiting for connection: %w", ctx.Err())
		case <-time.After(time.Duration(p.connTimeout) * time.Second):
		}
	}
}

// Close -.
func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
