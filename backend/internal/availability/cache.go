package availability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultSlotCacheTTL       = 5 * time.Minute
	defaultSlotCacheOpTimeout = 150 * time.Millisecond
	// initialSlotCacheVersion is used until a provider is first invalidated.
	initialSlotCacheVersion = "0"
)

// SlotCache caches computed slots in Redis under a per-provider version token.
//
// Keys: avail:ver:{providerID} holds an opaque random token; slots live at
// avail:slots:{providerID}:{version}:{YYYY-MM-DD}:{durationMinutes}. Invalidate
// writes a fresh token, so every older slot key becomes unreachable (it expires
// by TTL). Get returns the token it read and Set writes only under that token,
// so a fill that races an invalidation lands under a dead key. Every Redis call
// is bounded by opTimeout and every failure degrades to a cache miss.
type SlotCache struct {
	client    *redis.Client
	ttl       time.Duration
	opTimeout time.Duration
	logger    *slog.Logger
}

// NewSlotCache returns a cache over client; ttl <= 0 uses 5 minutes. A nil
// client yields a cache whose operations are no-ops returning a miss.
func NewSlotCache(client *redis.Client, ttl time.Duration) *SlotCache {
	if ttl <= 0 {
		ttl = defaultSlotCacheTTL
	}
	return &SlotCache{client: client, ttl: ttl, opTimeout: defaultSlotCacheOpTimeout}
}

// WithOpTimeout overrides the per-operation Redis timeout.
func (c *SlotCache) WithOpTimeout(d time.Duration) *SlotCache {
	if c != nil && d > 0 {
		c.opTimeout = d
	}
	return c
}

// WithLogger overrides the logger used for degraded-cache warnings.
func (c *SlotCache) WithLogger(l *slog.Logger) *SlotCache {
	if c != nil {
		c.logger = l
	}
	return c
}

func (c *SlotCache) enabled() bool { return c != nil && c.client != nil }

func (c *SlotCache) log() *slog.Logger {
	if c.logger != nil {
		return c.logger
	}
	return slog.Default()
}

func (c *SlotCache) opContext(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, c.opTimeout)
}

func versionKey(providerID string) string {
	return "avail:ver:" + providerID
}

func slotsKey(providerID, version string, date time.Time, durationMinutes int) string {
	return fmt.Sprintf("avail:slots:%s:%s:%s:%d", providerID, version, date.Format("2006-01-02"), durationMinutes)
}

// Get returns cached slots and the provider's current version token. On a
// miss the token is still returned (for Set); on any Redis error it returns
// ("", false) so the caller computes without caching.
func (c *SlotCache) Get(ctx context.Context, providerID string, date time.Time, durationMinutes int) (slots []TimeSlot, version string, hit bool) {
	if !c.enabled() {
		return nil, "", false
	}
	opCtx, cancel := c.opContext(ctx)
	defer cancel()

	version, err := c.client.Get(opCtx, versionKey(providerID)).Result()
	if errors.Is(err, redis.Nil) {
		version = initialSlotCacheVersion
	} else if err != nil {
		c.log().WarnContext(ctx, "slot cache read failed", "provider_id", providerID, "error", err)
		return nil, "", false
	}

	raw, err := c.client.Get(opCtx, slotsKey(providerID, version, date, durationMinutes)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, version, false
	}
	if err != nil {
		c.log().WarnContext(ctx, "slot cache read failed", "provider_id", providerID, "error", err)
		return nil, "", false
	}
	if err := json.Unmarshal(raw, &slots); err != nil {
		c.log().WarnContext(ctx, "slot cache decode failed", "provider_id", providerID, "error", err)
		return nil, version, false
	}
	return slots, version, true
}

// Set stores slots under the version token returned by Get. An empty version
// (Get failed) is a no-op.
func (c *SlotCache) Set(ctx context.Context, providerID string, date time.Time, durationMinutes int, version string, slots []TimeSlot) {
	if !c.enabled() || version == "" {
		return
	}
	payload, err := json.Marshal(slots)
	if err != nil {
		c.log().WarnContext(ctx, "slot cache encode failed", "provider_id", providerID, "error", err)
		return
	}
	opCtx, cancel := c.opContext(ctx)
	defer cancel()
	if err := c.client.Set(opCtx, slotsKey(providerID, version, date, durationMinutes), payload, c.ttl).Err(); err != nil {
		c.log().WarnContext(ctx, "slot cache write failed", "provider_id", providerID, "error", err)
	}
}

// Invalidate makes every cached slot list for the provider unreachable by
// writing a new random version token.
func (c *SlotCache) Invalidate(ctx context.Context, providerID string) {
	if !c.enabled() {
		return
	}
	token, err := newVersionToken()
	if err != nil {
		c.log().WarnContext(ctx, "slot cache invalidate failed", "provider_id", providerID, "error", err)
		return
	}
	opCtx, cancel := c.opContext(ctx)
	defer cancel()
	if err := c.client.Set(opCtx, versionKey(providerID), token, 0).Err(); err != nil {
		c.log().WarnContext(ctx, "slot cache invalidate failed", "provider_id", providerID, "error", err)
	}
}

func newVersionToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate version token: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
