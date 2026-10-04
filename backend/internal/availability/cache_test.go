package availability

import (
	"bytes"
	"context"
	"log/slog"
	"net"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"temu.in/backend/internal/cache"
	"temu.in/backend/internal/testutil"
)

var cacheDate = time.Date(2030, 3, 4, 0, 0, 0, 0, time.UTC)

func sampleSlots() []TimeSlot {
	return []TimeSlot{
		{Date: "2030-03-04", StartTime: "09:00", EndTime: "10:00"},
		{Date: "2030-03-04", StartTime: "10:00", EndTime: "11:00"},
	}
}

func TestSlotCacheMissThenHit(t *testing.T) {
	client, _ := testutil.Redis(t)
	c := NewSlotCache(client, 0)
	ctx := context.Background()

	slots, version, hit := c.Get(ctx, "p1", cacheDate, 60)
	if hit || slots != nil || version != "0" {
		t.Fatalf("first Get = %v, %q, %v; want miss with version 0", slots, version, hit)
	}
	c.Set(ctx, "p1", cacheDate, 60, version, sampleSlots())

	slots, version2, hit := c.Get(ctx, "p1", cacheDate, 60)
	if !hit || version2 != version || !reflect.DeepEqual(slots, sampleSlots()) {
		t.Fatalf("second Get = %v, %q, %v", slots, version2, hit)
	}

	if _, _, hit := c.Get(ctx, "p1", cacheDate, 30); hit {
		t.Fatal("different duration must be a separate key")
	}
	if _, _, hit := c.Get(ctx, "p1", cacheDate.AddDate(0, 0, 1), 60); hit {
		t.Fatal("different date must be a separate key")
	}
	if _, _, hit := c.Get(ctx, "p2", cacheDate, 60); hit {
		t.Fatal("different provider must be a separate key")
	}
}

func TestSlotCacheEmptySlotsRoundTrip(t *testing.T) {
	client, _ := testutil.Redis(t)
	c := NewSlotCache(client, 0)
	ctx := context.Background()
	_, v, _ := c.Get(ctx, "p1", cacheDate, 60)
	c.Set(ctx, "p1", cacheDate, 60, v, []TimeSlot{})
	slots, _, hit := c.Get(ctx, "p1", cacheDate, 60)
	if !hit || slots == nil || len(slots) != 0 {
		t.Fatalf("empty slots = %#v, hit=%v", slots, hit)
	}
}

func TestSlotCacheTTL(t *testing.T) {
	client, s := testutil.Redis(t)
	c := NewSlotCache(client, 0)
	ctx := context.Background()
	_, v, _ := c.Get(ctx, "p1", cacheDate, 60)
	c.Set(ctx, "p1", cacheDate, 60, v, sampleSlots())

	key := slotsKey("p1", v, cacheDate, 60)
	if ttl := s.TTL(key); ttl < 4*time.Minute || ttl > 5*time.Minute {
		t.Fatalf("TTL = %v, want ~5m", ttl)
	}
	s.FastForward(5*time.Minute + time.Second)
	if _, _, hit := c.Get(ctx, "p1", cacheDate, 60); hit {
		t.Fatal("expired entry must miss")
	}
}

func TestSlotCacheInvalidate(t *testing.T) {
	client, s := testutil.Redis(t)
	c := NewSlotCache(client, 0)
	ctx := context.Background()
	_, v1, _ := c.Get(ctx, "p1", cacheDate, 60)
	c.Set(ctx, "p1", cacheDate, 60, v1, sampleSlots())
	c.Set(ctx, "p2", cacheDate, 60, v1, sampleSlots())

	c.Invalidate(ctx, "p1")
	slots, v2, hit := c.Get(ctx, "p1", cacheDate, 60)
	if hit || slots != nil {
		t.Fatal("invalidated provider must miss")
	}
	if v2 == v1 || len(v2) != 32 {
		t.Fatalf("new token %q must differ from %q and be 32 hex chars", v2, v1)
	}
	if _, _, hit := c.Get(ctx, "p2", cacheDate, 60); !hit {
		t.Fatal("other providers must stay cached")
	}

	c.Invalidate(ctx, "p1")
	v3, _ := s.Get(versionKey("p1"))
	if v3 == v2 {
		t.Fatal("each invalidation must write a fresh token")
	}
	if ttl := s.TTL(versionKey("p1")); ttl != 0 {
		t.Fatalf("version key must have no TTL, got %v", ttl)
	}
}

func TestSlotCacheStaleFillRace(t *testing.T) {
	client, _ := testutil.Redis(t)
	c := NewSlotCache(client, 0)
	ctx := context.Background()

	_, versionA, hit := c.Get(ctx, "p1", cacheDate, 60)
	if hit {
		t.Fatal("expected miss")
	}
	c.Invalidate(ctx, "p1")
	c.Set(ctx, "p1", cacheDate, 60, versionA, sampleSlots()) // computed before invalidation
	if _, _, hit := c.Get(ctx, "p1", cacheDate, 60); hit {
		t.Fatal("stale fill under the old version must not be readable")
	}
}

func TestSlotCacheSetEmptyVersionIsNoop(t *testing.T) {
	client, s := testutil.Redis(t)
	c := NewSlotCache(client, 0)
	c.Set(context.Background(), "p1", cacheDate, 60, "", sampleSlots())
	if keys := s.Keys(); len(keys) != 0 {
		t.Fatalf("expected no keys, got %v", keys)
	}
}

func TestSlotCacheNilIsNoop(t *testing.T) {
	ctx := context.Background()
	for _, c := range []*SlotCache{nil, NewSlotCache(nil, 0)} {
		if slots, v, hit := c.Get(ctx, "p1", cacheDate, 60); hit || slots != nil || v != "" {
			t.Fatal("nil cache must miss")
		}
		c.Set(ctx, "p1", cacheDate, 60, "0", sampleSlots())
		c.Invalidate(ctx, "p1")
	}
}

func TestSlotCacheClosedRedisDegrades(t *testing.T) {
	client, s := testutil.Redis(t)
	var buf bytes.Buffer
	c := NewSlotCache(client, 0).WithLogger(slog.New(slog.NewTextHandler(&buf, nil)))
	s.Close()
	ctx := context.Background()

	if slots, v, hit := c.Get(ctx, "p1", cacheDate, 60); hit || slots != nil || v != "" {
		t.Fatalf("Get on closed redis = %v, %q, %v", slots, v, hit)
	}
	c.Set(ctx, "p1", cacheDate, 60, "0", sampleSlots())
	c.Invalidate(ctx, "p1")

	out := buf.String()
	for _, msg := range []string{"slot cache read failed", "slot cache write failed", "slot cache invalidate failed"} {
		if !strings.Contains(out, msg) {
			t.Errorf("missing warning %q in %q", msg, out)
		}
	}
	if !strings.Contains(out, "level=WARN") {
		t.Errorf("expected WARN level: %q", out)
	}
}

// hungListener accepts connections and never replies.
func hungListener(t *testing.T) net.Listener {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	var mu sync.Mutex
	var conns []net.Conn
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		_ = l.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			_ = c.Close()
		}
	})
	return l
}

func TestSlotCacheHungRedisRespectsOpTimeout(t *testing.T) {
	l := hungListener(t)
	client := cache.NewClientFromOptions(&redis.Options{Addr: l.Addr().String()})
	t.Cleanup(func() { _ = client.Close() })

	var buf bytes.Buffer
	c := NewSlotCache(client, 0).WithOpTimeout(50 * time.Millisecond).WithLogger(slog.New(slog.NewTextHandler(&buf, nil)))
	ctx := context.Background()

	check := func(name string, op func()) {
		t.Helper()
		start := time.Now()
		op()
		if elapsed := time.Since(start); elapsed >= 500*time.Millisecond {
			t.Errorf("%s took %v, want < 500ms", name, elapsed)
		}
	}
	check("Get", func() {
		if _, _, hit := c.Get(ctx, "p1", cacheDate, 60); hit {
			t.Error("hung Get must miss")
		}
	})
	check("Set", func() { c.Set(ctx, "p1", cacheDate, 60, "0", sampleSlots()) })
	check("Invalidate", func() { c.Invalidate(ctx, "p1") })

	if n := strings.Count(buf.String(), "level=WARN"); n < 3 {
		t.Fatalf("expected 3 warnings, got %d: %q", n, buf.String())
	}
}
