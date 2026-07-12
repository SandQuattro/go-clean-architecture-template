package app

import (
	"clean-arch-template/config"
	"clean-arch-template/pkg/logger"
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

const (
	defaultAttempts = 3
	defaultTimeout  = time.Second
)

// applyMigrations применяет миграции при старте через goose. Любая ошибка
// возвращается наверх — сервис не должен принимать трафик на битой схеме.
// Session-lock (pg advisory lock) защищает от параллельного применения
// несколькими репликами. В продакшене предпочтителен отдельный Job/initContainer.
func applyMigrations(ctx context.Context, cfg config.DB, log logger.Logger) error {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return fmt.Errorf("migrate: open db: %w", err)
	}
	defer func() {
		if cerr := db.Close(); cerr != nil {
			log.Error(ctx, "migrate: close db", "error", cerr.Error())
		}
	}()

	for attempts := defaultAttempts; attempts > 0; attempts-- {
		err = db.PingContext(ctx)
		if err == nil {
			break
		}
		log.Debug(ctx, fmt.Sprintf("migrate: postgres is trying to connect, attempts left: %d", attempts))
		time.Sleep(defaultTimeout)
	}
	if err != nil {
		return fmt.Errorf("migrate: postgres connect: %w", err)
	}

	sessionLocker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("migrate: session locker: %w", err)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		os.DirFS(cfg.MigrationsDir),
		goose.WithSessionLocker(sessionLocker),
	)
	if err != nil {
		return fmt.Errorf("migrate: provider: %w", err)
	}
	defer func() {
		if cerr := provider.Close(); cerr != nil {
			log.Error(ctx, "migrate: close provider", "error", cerr.Error())
		}
	}()

	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate: up: %w", err)
	}

	if len(results) == 0 {
		log.Info(ctx, "Migrate: no change")
		return nil
	}

	log.Info(ctx, fmt.Sprintf("Migrate: applied %d migrations", len(results)))

	return nil
}
