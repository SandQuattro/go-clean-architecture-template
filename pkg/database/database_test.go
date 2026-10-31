package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// unreachableDSN указывает на закрытый порт: подключение отклоняется сразу.
const unreachableDSN = "postgres://user:password@127.0.0.1:1/demo?sslmode=disable"

func TestNewFailsWhenDatabaseIsUnreachable(t *testing.T) {
	t.Parallel()

	pg, err := New(context.Background(), unreachableDSN, ConnAttempts(1), ConnTimeout(1))

	require.Error(t, err)
	require.Nil(t, pg)
}

func TestNewStopsWaitingOnContextCancel(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	pg, err := New(ctx, unreachableDSN, ConnAttempts(3), ConnTimeout(5))

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Nil(t, pg)
	require.Less(t, time.Since(start), 5*time.Second,
		"отмена контекста должна прерывать паузу между попытками, а не пересиживать её")
}
