package database

import (
	"clean-arch-template/config"
	"clean-arch-template/pkg/logger"
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/multitracer"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupPoolConfig(cfg *config.Config, pg *Postgres, poolConfig *pgxpool.Config) {
	poolConfig.MinConns = pg.minPoolSize // Keep warm connections for health checks and latency spikes
	poolConfig.MaxConns = pg.maxPoolSize
	poolConfig.ConnConfig.ConnectTimeout = time.Duration(pg.connTimeout) * time.Second
	poolConfig.HealthCheckPeriod = time.Duration(pg.healthCheckPeriod) * time.Minute

	if cfg.Debug {
		poolConfig.ConnConfig.Tracer = &multitracer.Tracer{
			QueryTracers: []pgx.QueryTracer{
				&SQLQueryTracer{log: pg.logger},
			},
			ConnectTracers: []pgx.ConnectTracer{
				&ConnectTracer{log: pg.logger},
			},
		}

		poolConfig.PrepareConn = func(ctx context.Context, conn *pgx.Conn) (bool, error) {
			pg.logger.Debug(ctx, "[PGPOOL] attempting to acquire connection")
			return true, nil
		}

		poolConfig.AfterRelease = func(conn *pgx.Conn) bool {
			pg.logger.Debug(context.Background(), "[PGPOOL] connection released to pool")
			return true
		}

		// Add health check logging
		poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
			pg.logger.Debug(ctx, "[PGPOOL] health check: new connection established")
			return nil
		}

		// Log when connection is being checked
		poolConfig.BeforeClose = func(conn *pgx.Conn) {
			pg.logger.Debug(context.Background(), "[PGPOOL] health check: connection being closed")
		}
	}
}

type SQLQueryTracer struct {
	log logger.Logger
}

func (t *SQLQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	// pgx uses SELECT 1 for health checks
	if data.SQL == "SELECT 1" {
		t.log.Debug(ctx, "[PGPOOL] Health check ping started",
			"conn_id", fmt.Sprintf("%p", conn),
			"pid", conn.PgConn().PID(),
		)
	} else {
		t.log.Debug(ctx, "[PGPOOL] SQL query started", "sql", data.SQL, "args", fmt.Sprintf("%v", data.Args))
	}
	return ctx
}

// TraceQueryEnd игнорирует входящий ctx: к завершению запроса он уже может
// быть отменён (drain/таймаут), поэтому лог пишется с context.Background().
func (t *SQLQueryTracer) TraceQueryEnd(_ context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	if data.Err != nil {
		//nolint:contextcheck // см. комментарий к TraceQueryEnd
		t.log.Error(context.Background(), "[PGPOOL] SQL query failed", "error", data.Err)
		return
	}
	// Check if this was a health check ping (CommandTag will be "SELECT 1")
	if data.CommandTag.String() == "SELECT 1" {
		//nolint:contextcheck // см. комментарий к TraceQueryEnd
		t.log.Debug(context.Background(), "[PGPOOL] Health check ping completed",
			"conn_id", fmt.Sprintf("%p", conn),
			"pid", conn.PgConn().PID(),
			"command", data.CommandTag,
		)
	} else {
		//nolint:contextcheck // см. комментарий к TraceQueryEnd
		t.log.Debug(context.Background(), "[PGPOOL] SQL query completed", "rows_affected", data.CommandTag.RowsAffected())
	}
}

type ConnectTracer struct {
	log logger.Logger
}

func (t *ConnectTracer) TraceConnectStart(ctx context.Context, data pgx.TraceConnectStartData) context.Context {
	t.log.Debug(ctx, "[PGPOOL] Connecting to database")
	return ctx
}

// TraceConnectEnd игнорирует входящий ctx: см. комментарий к TraceQueryEnd.
func (t *ConnectTracer) TraceConnectEnd(_ context.Context, data pgx.TraceConnectEndData) {
	if data.Err != nil {
		//nolint:contextcheck // см. комментарий к TraceConnectEnd
		t.log.Error(context.Background(), "[PGPOOL] Connecting to database failed", "error", data.Err)
		return
	}
	//nolint:contextcheck // см. комментарий к TraceConnectEnd
	t.log.Debug(context.Background(), "[PGPOOL] Connected to database")
}
