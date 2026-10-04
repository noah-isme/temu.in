package testutil

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"temu.in/backend/internal/cache"
)

// Redis starts an in-process miniredis server for the test and returns a
// client built with the production constructor plus the server handle (for
// FastForward, Close, and key inspection). Both are cleaned up with the test.
func Redis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	s := miniredis.RunT(t)
	client := cache.NewClientFromOptions(&redis.Options{Addr: s.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client, s
}
