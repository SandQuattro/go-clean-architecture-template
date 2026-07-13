package app

import (
	"context"
	"testing"
	"time"

	"clean-arch-template/config"
	"clean-arch-template/pkg/logger/loggertest"

	"github.com/stretchr/testify/require"
)

func TestApplyMigrationsStopsOnCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := config.DB{
		DBHost:        "127.0.0.1",
		DBPort:        1, // закрытый порт: ping гарантированно не проходит
		DBUser:        "user",
		DBPassword:    "password",
		DBName:        "demo",
		SSLMode:       "disable",
		MigrationsDir: "migrations",
	}

	start := time.Now()
	err := applyMigrations(ctx, cfg, &loggertest.Fake{})

	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, time.Since(start), defaultTimeout,
		"отменённый контекст должен прерывать retry-паузы, а не пересиживать их")
}
