package app

import (
	"clean-arch-template/config"
	"clean-arch-template/pkg/logger/loggertest"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Отменённый контекст должен доходить до goose: подключение и ожидание
// advisory lock прерываются, а не висят. Без проброса ctx тест получил бы
// ошибку подключения к закрытому порту вместо context.Canceled.
func TestApplyMigrationsStopsOnCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := config.DB{
		DBHost:        "127.0.0.1",
		DBPort:        1, // закрытый порт: подключение гарантированно не проходит
		DBUser:        "user",
		DBPassword:    "password",
		DBName:        "demo",
		SSLMode:       "disable",
		MigrationsDir: "../../migrations",
	}

	start := time.Now()
	err := applyMigrations(ctx, cfg, &loggertest.Fake{})

	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, time.Since(start), time.Second,
		"отменённый контекст должен прерывать применение миграций сразу")
}
