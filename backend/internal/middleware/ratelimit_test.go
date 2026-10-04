package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"temu.in/backend/internal/testutil"
)

func limitedEngine(mw gin.HandlerFunc) *gin.Engine {
	engine := gin.New()
	engine.Use(mw)
	engine.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	return engine
}

func hit(engine *gin.Engine) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.RemoteAddr = "203.0.113.7:1234"
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestRateLimiterUnderAndOverLimit(t *testing.T) {
	client, mr := testutil.Redis(t)
	engine := limitedEngine(NewRateLimiter(client, 2, time.Minute).Middleware())

	first := hit(engine)
	if first.Code != http.StatusOK || first.Header().Get("X-RateLimit-Limit") != "2" || first.Header().Get("X-RateLimit-Remaining") != "1" {
		t.Fatalf("first: %d %v", first.Code, first.Header())
	}
	if ttl := mr.TTL("rl:203.0.113.7"); ttl <= 0 || ttl > time.Minute {
		t.Fatalf("window TTL = %v", ttl)
	}
	if second := hit(engine); second.Code != http.StatusOK || second.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Fatalf("second: %d %v", second.Code, second.Header())
	}
	third := hit(engine)
	if third.Code != http.StatusTooManyRequests || third.Header().Get("Retry-After") != "60" {
		t.Fatalf("third: %d %v", third.Code, third.Header())
	}

	// The window expiring resets the counter.
	mr.FastForward(time.Minute + time.Second)
	if again := hit(engine); again.Code != http.StatusOK {
		t.Fatalf("after window: %d", again.Code)
	}
}

func TestRateLimiterNilCachePassesThrough(t *testing.T) {
	engine := limitedEngine(NewRateLimiter(nil, 1, time.Minute).Middleware())
	for i := 0; i < 3; i++ {
		if w := hit(engine); w.Code != http.StatusOK {
			t.Fatalf("request %d: %d", i, w.Code)
		}
	}
	routeEngine := limitedEngine(RateLimitByRoute(nil, 1, time.Minute))
	if w := hit(routeEngine); w.Code != http.StatusOK {
		t.Fatalf("route limiter with nil cache: %d", w.Code)
	}
}

func TestRateLimiterFailsOpenWhenRedisIsDown(t *testing.T) {
	client, mr := testutil.Redis(t)
	mr.Close()
	engine := limitedEngine(NewRateLimiter(client, 1, time.Minute).Middleware())
	for i := 0; i < 3; i++ {
		if w := hit(engine); w.Code != http.StatusOK {
			t.Fatalf("request %d with Redis down: %d", i, w.Code)
		}
	}
	closed := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	t.Cleanup(func() { _ = closed.Close() })
	if w := hit(limitedEngine(RateLimitByRoute(closed, 1, time.Minute))); w.Code != http.StatusOK {
		t.Fatalf("route limiter with Redis down: %d", w.Code)
	}
}

func TestRateLimitByRoute(t *testing.T) {
	client, _ := testutil.Redis(t)
	engine := limitedEngine(RateLimitByRoute(client, 1, time.Minute))
	if w := hit(engine); w.Code != http.StatusOK {
		t.Fatalf("first: %d", w.Code)
	}
	w := hit(engine)
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") != "60" {
		t.Fatalf("second: %d %v", w.Code, w.Header())
	}
}
