package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"temu.in/backend/internal/cache"
	"temu.in/backend/internal/config"
	"temu.in/backend/internal/database"
	"temu.in/backend/internal/middleware"
	"temu.in/backend/internal/router"
	"temu.in/backend/internal/seeder"
)

type Server struct {
	cfg    *config.Config
	router *gin.Engine
	db     *gorm.DB
	cache  *redis.Client
}

func New(cfg *config.Config) *Server {
	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.AccessLog(slog.Default()),
		gin.CustomRecovery(func(c *gin.Context, err any) {
			// debug.Stack runs inside the recovery callback, so it is the panicking goroutine's stack.
			slog.Error("panic recovered",
				"error", fmt.Sprint(err),
				"stack", string(debug.Stack()),
				"request_id", middleware.RequestIDFromGin(c),
				"method", c.Request.Method,
				"path", c.Request.URL.Path,
			)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		}),
	)
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", middleware.RequestIDHeader},
		ExposeHeaders:    []string{middleware.RequestIDHeader},
		AllowCredentials: true,
		MaxAge:           12 * 60 * 60, // 12 hours
	}))

	srv := &Server{cfg: cfg, router: r}
	return srv
}

func (s *Server) connectRedis() error {
	client, err := cache.NewClient(s.cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("create redis client: %w", err)
	}

	s.cache = client

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := s.cache.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("ping redis: %w", err)
	}

	return nil
}

func (s *Server) DB() *gorm.DB {
	return s.db
}

func (s *Server) Cache() *redis.Client {
	return s.cache
}

func (s *Server) Run() error {
	if err := s.connectDatabase(); err != nil {
		return err
	}

	if err := s.connectRedis(); err != nil {
		return err
	}

	rateLimiter := middleware.NewRateLimiter(s.cache, s.cfg.RateLimitRequests, s.cfg.RateLimitWindow)
	// Orchestrator probes must never be 429'd.
	s.router.Use(middleware.SkipPaths(middleware.HealthPathPrefixes, rateLimiter.Middleware()))

	r := router.New(s.router, s.cfg, s.db, s.cache)
	r.RegisterAll()
	workerContext, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	r.StartBackgroundWorkers(workerContext)

	addr := fmt.Sprintf(":%d", s.cfg.Port)
	return s.router.Run(addr)
}

func (s *Server) connectDatabase() error {
	db, err := database.Connect(s.cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	if err := db.RunMigrations(); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	s.db = db.DB

	if err := seeder.SeedAdmin(s.db); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}

	return nil
}
