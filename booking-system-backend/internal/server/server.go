package server

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/temu-in/temu.in/booking-system-backend/internal/config"
	"github.com/temu-in/temu.in/booking-system-backend/internal/health"
	"github.com/temu-in/temu.in/booking-system-backend/internal/models"
	authhandler "github.com/temu-in/temu.in/booking-system-backend/internal/auth"
	"github.com/temu-in/temu.in/booking-system-backend/internal/user"
	"github.com/temu-in/temu.in/booking-system-backend/internal/seeder"
)

type Server struct {
	cfg    *config.Config
	router *gin.Engine
	db     *gorm.DB
	cache  *redis.Client
}

func New(cfg *config.Config) *Server {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization"},
	}))

	srv := &Server{cfg: cfg, router: r}
	srv.registerRoutes()
	return srv
}

func (s *Server) registerRoutes() {
	health.RegisterRoutes(s.router)
}

func (s *Server) Run() error {
	if err := s.connectDatabase(); err != nil {
		return err
	}

	if err := s.connectRedis(); err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", s.cfg.Port)
	return s.router.Run(addr)
}

func (s *Server) connectDatabase() error {
	db, err := gorm.Open(postgres.Open(s.cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	s.db = db
	// auto-migrate core models
	if err := s.db.AutoMigrate(&models.User{}); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	// register auth routes after DB connected
	repo := user.NewRepository(s.db)
	h := authhandler.NewHandler(repo, s.cfg)
	api := s.router.Group("/api")
	h.RegisterRoutes(api.Group("/auth"))

	// seed admin if requested
	if err := seeder.SeedAdmin(s.db); err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}

	// register /api/me
	userHandler := user.NewHandler(repo)
	userHandler.RegisterRoutes(api.Group("/"), s.cfg.JWTSecret)

	// sample admin-only route
	adminGroup := api.Group("/admin")
	adminGroup.Use(authhandler.Middleware(s.cfg.JWTSecret), authhandler.RequireRole("admin"))
	adminGroup.GET("/stats", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok", "users": 42}) })

	return nil
}

func (s *Server) connectRedis() error {
	options, err := redis.ParseURL(s.cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("parse redis url: %w", err)
	}

	s.cache = redis.NewClient(options)

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

func (s *Server) Router() *gin.Engine {
	return s.router
}
