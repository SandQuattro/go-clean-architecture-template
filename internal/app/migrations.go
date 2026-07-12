package app

import (
	"clean-arch-template/config"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	defaultAttempts = 3
	defaultTimeout  = time.Second
)

// applyMigrations применяет миграции при старте. Любая ошибка (включая dirty
// state) возвращается наверх — сервис не должен принимать трафик на битой схеме.
// В продакшене с несколькими репликами предпочтителен отдельный Job/initContainer.
func applyMigrations(cfg config.DB) error {
	var (
		attempts = defaultAttempts
		err      error
		m        *migrate.Migrate
	)

	for attempts > 0 {
		m, err = migrate.New("file://migrations", cfg.DSN())
		if err == nil {
			break
		}

		slog.Debug(fmt.Sprintf("migrate: postgres is trying to connect, attempts left: %d", attempts))
		time.Sleep(defaultTimeout)

		attempts--
	}

	if err != nil {
		return fmt.Errorf("migrate: postgres connect: %w", err)
	}

	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			slog.Error(fmt.Sprintf("migrate: close source: %s", srcErr))
		}
		if dbErr != nil {
			slog.Error(fmt.Sprintf("migrate: close db conn: %s", dbErr))
		}
	}()

	err = m.Up()

	switch {
	case errors.Is(err, migrate.ErrNoChange):
		slog.Info("Migrate: no change")
		return nil
	case err != nil:
		return fmt.Errorf("migrate: up: %w", err)
	}

	slog.Info("Migrate: up success")

	return nil
}
