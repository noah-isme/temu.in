// Package cache builds Redis clients with the settings the application relies on.
package cache

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	dialTimeout = 2 * time.Second
	ioTimeout   = 1 * time.Second
	maxRetries  = 1
)

// NewClient parses redisURL and returns a client whose commands honour the
// caller's context deadline (ContextTimeoutEnabled), so a hung Redis cannot
// block a request beyond the deadline the caller chose.
func NewClient(redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	return NewClientFromOptions(opts), nil
}

// NewClientFromOptions applies the shared defaults to opts and returns a client.
func NewClientFromOptions(opts *redis.Options) *redis.Client {
	applyDefaults(opts)
	return redis.NewClient(opts)
}

func applyDefaults(opts *redis.Options) {
	opts.ContextTimeoutEnabled = true
	opts.DialTimeout = dialTimeout
	opts.ReadTimeout = ioTimeout
	opts.WriteTimeout = ioTimeout
	opts.MaxRetries = maxRetries
}
