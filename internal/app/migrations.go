package app

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"clean-arch-template/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	defaultAttempts = 3
	defaultTimeout  = time.Second
)

func applyMigrations(cfg config.DB) error {
	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	var (
		attempts = defaultAttempts
		err      error
		m        *migrate.Migrate
	)

	for attempts > 0 {
		m, err = migrate.New("file://migrations", databaseURL)
		if err == nil {
			break
		}

		slog.Debug(fmt.Sprintf("migrate: postgres is trying to connect, attempts left: %d", attempts))
		time.Sleep(defaultTimeout)

		attempts--
	}

	if err != nil {
		slog.Error(fmt.Sprintf("migrate: postgres connect error: %s", err))
		return err
	}

	err = m.Up()

	defer func() {
		err1, err2 := m.Close()
		if err1 != nil {
			slog.Error(fmt.Sprintf("failed close db conn: %s", err1.Error()))
		}

		if err2 != nil {
			slog.Error(fmt.Sprintf("failed close db conn: %s", err2.Error()))
		}
	}()

	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		slog.Error(fmt.Sprintf("Migrate: up error: %s", err))
	}

	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("Migrate: no change")
		return nil
	}

	slog.Info("Migrate: up success")

	return nil
}
