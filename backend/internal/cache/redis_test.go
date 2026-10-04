package cache

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// hangingListener accepts connections and never replies, simulating a hung Redis.
func hangingListener(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	var (
		mu    sync.Mutex
		conns []net.Conn
	)
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, c)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			_ = c.Close()
		}
	})
	return ln.Addr().String()
}

func TestNewClientAppliesDefaults(t *testing.T) {
	c, err := NewClient("redis://:secret@localhost:6379/2")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()
	o := c.Options()
	if !o.ContextTimeoutEnabled {
		t.Error("ContextTimeoutEnabled not set")
	}
	if o.DialTimeout != dialTimeout || o.ReadTimeout != ioTimeout || o.WriteTimeout != ioTimeout {
		t.Errorf("timeouts = %v/%v/%v", o.DialTimeout, o.ReadTimeout, o.WriteTimeout)
	}
	if o.MaxRetries != maxRetries {
		t.Errorf("MaxRetries = %d", o.MaxRetries)
	}
	if o.DB != 2 || o.Password != "secret" {
		t.Errorf("URL settings lost: db=%d", o.DB)
	}
}

func TestNewClientInvalidURL(t *testing.T) {
	if _, err := NewClient("not-a-url://x"); err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestClientHonoursContextDeadlineOnHungServer(t *testing.T) {
	addr := hangingListener(t)
	c := NewClientFromOptions(&redis.Options{Addr: addr})
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := c.Get(ctx, "k").Err()
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from hung server")
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("Get took %v, want < 500ms (context deadline ignored)", elapsed)
	}
}
