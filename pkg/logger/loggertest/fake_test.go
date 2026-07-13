package loggertest

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFakeWithRecordsInheritedArgs(t *testing.T) {
	t.Parallel()

	fake := &Fake{}
	child := fake.With("component", "test").With("shard", 1)

	child.Info(context.Background(), "hello", "user_id", 42)

	require.Len(t, fake.Entries, 1)
	assert.Equal(t, Entry{
		Level: "INFO",
		Msg:   "hello",
		Args:  []any{"component", "test", "shard", 1, "user_id", 42},
	}, fake.Entries[0])
}

func TestFakeRecordsAllLevels(t *testing.T) {
	t.Parallel()

	fake := &Fake{}
	ctx := context.Background()

	fake.Debug(ctx, "d")
	fake.Info(ctx, "i")
	fake.Warn(ctx, "w")
	fake.Error(ctx, "e")

	require.Len(t, fake.Entries, 4)
	assert.Equal(t, "DEBUG", fake.Entries[0].Level)
	assert.Equal(t, "INFO", fake.Entries[1].Level)
	assert.Equal(t, "WARN", fake.Entries[2].Level)
	assert.Equal(t, "ERROR", fake.Entries[3].Level)
}

func TestFakeConcurrentAccess(t *testing.T) {
	t.Parallel()

	const (
		writers   = 8
		perWriter = 50
	)

	fake := &Fake{}

	var wg sync.WaitGroup
	for range writers {
		wg.Go(func() {
			log := fake.With("writer", true)
			for range perWriter {
				log.Info(context.Background(), "msg")
			}
		})
	}
	wg.Wait()

	assert.Len(t, fake.Entries, writers*perWriter)
}
