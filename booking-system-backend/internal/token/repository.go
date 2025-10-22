package token

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"

    "github.com/temu-in/temu.in/booking-system-backend/internal/models"
)

type Repository struct{
    db *gorm.DB
    cache *redis.Client // optional redis client for blacklist / notifications
}

func NewRepository(db *gorm.DB) *Repository {
    return &Repository{db: db}
}

// NewRepositoryWithCache creates a token repository with an attached Redis client
func NewRepositoryWithCache(db *gorm.DB, cache *redis.Client) *Repository {
    return &Repository{db: db, cache: cache}
}

func (r *Repository) Create(t *models.RefreshToken) error {
    return r.db.Create(t).Error
}

func (r *Repository) RevokeByHash(hash string) error {
    if err := r.db.Model(&models.RefreshToken{}).Where("token_hash = ?", hash).Update("revoked", true).Error; err != nil {
        return err
    }
    // add to redis blacklist for fast checks (set key with TTL equal to token expiry window)
    if r.cache != nil {
        ctx := context.Background()
        key := fmt.Sprintf("revoked_rt:%s", hash)
        // store a simple value; set TTL to 30 days as a conservative upper bound
        _ = r.cache.Set(ctx, key, "1", 30*24*time.Hour).Err()
        // publish notification for other instances (best-effort)
        _ = r.cache.Publish(ctx, "revoked_refresh", hash).Err()
    }
    return nil
}

func (r *Repository) FindByHash(hash string) (*models.RefreshToken, error) {
    // quick check in redis blacklist
    if r.cache != nil {
        ctx := context.Background()
        key := fmt.Sprintf("revoked_rt:%s", hash)
        if exists, _ := r.cache.Exists(ctx, key).Result(); exists > 0 {
            // treat as not found / revoked
            return nil, gorm.ErrRecordNotFound
        }
    }

    var t models.RefreshToken
    if err := r.db.Where("token_hash = ?", hash).First(&t).Error; err != nil {
        return nil, err
    }
    return &t, nil
}

func (r *Repository) RevokeAllForUser(userID uint) error {
    if err := r.db.Model(&models.RefreshToken{}).Where("user_id = ?", userID).Update("revoked", true).Error; err != nil {
        return err
    }
    // optionally publish a pattern key for all tokens for the user
    if r.cache != nil {
        ctx := context.Background()
        _ = r.cache.Publish(ctx, "revoked_refresh_user", fmt.Sprintf("user:%d", userID)).Err()
    }
    return nil
}
