package router

import (
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"temu.in/backend/internal/admin"
	"temu.in/backend/internal/audit"
	authhandler "temu.in/backend/internal/auth"
	"temu.in/backend/internal/availability"
	"temu.in/backend/internal/booking"
	"temu.in/backend/internal/config"
	"temu.in/backend/internal/favorite"
	"temu.in/backend/internal/health"
	"temu.in/backend/internal/notification"
	"temu.in/backend/internal/payment"
	"temu.in/backend/internal/provider"
	"temu.in/backend/internal/public"
	"temu.in/backend/internal/review"
	"temu.in/backend/internal/service"
	"temu.in/backend/internal/token"
	"temu.in/backend/internal/user"
)

type Router struct {
	engine *gin.Engine
	cfg    *config.Config
	db     *gorm.DB
	cache  *redis.Client
}

func New(engine *gin.Engine, cfg *config.Config, db *gorm.DB, cache *redis.Client) *Router {
	return &Router{
		engine: engine,
		cfg:    cfg,
		db:     db,
		cache:  cache,
	}
}

func (r *Router) RegisterAll() {
	r.registerHealth()
	r.registerPublic()
	r.registerAuth()
	r.registerProviders()
	r.registerServices()
	r.registerAvailability()
	r.registerReviews()
	r.registerAdmin()
	r.registerBookings()
	r.registerPayments()
	r.registerFavorites()
}

func (r *Router) registerHealth() {
	health.RegisterRoutes(r.engine)
	apiGroup := r.engine.Group("/api")
	apiGroup.GET("/health/live", func(c *gin.Context) { c.JSON(200, gin.H{"status": "live"}) })
	apiGroup.GET("/health/ready", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ready"}) })
}

func (r *Router) registerPublic() {
	public.RegisterRoutes(r.engine)
	public.RegisterRoutes(r.engine.Group("/api"))
}

func (r *Router) registerAuth() {
	repo := user.NewRepository(r.db)
	var tokenRepo *token.Repository
	if r.cache != nil {
		tokenRepo = token.NewRepositoryWithCache(r.db, r.cache)
	} else {
		tokenRepo = token.NewRepository(r.db)
	}

	emailProvider := notification.NewConsoleEmailProvider()
	notificationSvc := notification.NewService(emailProvider)

	handler := authhandler.NewHandler(repo, r.cfg, tokenRepo, notificationSvc)
	api := r.engine.Group("/api")
	handler.RegisterRoutes(api.Group("/auth"))

	api.GET("/me", authhandler.Middleware(r.cfg.JWTSecret), func(c *gin.Context) {
		v, ok := c.Get(authhandler.UserContextKey)
		if !ok {
			c.JSON(401, gin.H{"error": "unauthenticated"})
			return
		}
		claims, ok := v.(*authhandler.Claims)
		if !ok {
			c.JSON(500, gin.H{"error": "internal"})
			return
		}
		u, err := repo.FindByID(claims.UserID)
		if err != nil || u == nil {
			c.JSON(500, gin.H{"error": "internal"})
			return
		}
		c.JSON(200, gin.H{"user": gin.H{"id": u.ID, "email": u.Email, "role": u.Role, "name": u.Name}})
	})
}

func (r *Router) registerAdmin() {
	repo := user.NewRepository(r.db)
	auditRepo := audit.NewRepository(r.db)
	handler := admin.NewHandler(repo, auditRepo)

	api := r.engine.Group("/api")
	handler.RegisterRoutes(api.Group("/"), r.cfg.JWTSecret)

	adminGroup := api.Group("/admin")
	adminGroup.Use(authhandler.Middleware(r.cfg.JWTSecret), authhandler.RequireRole("admin"))
	adminGroup.GET("/stats", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok", "users": 42}) })
}

func (r *Router) registerBookings() {
	bookingRepo := booking.NewRepository(r.db)
	paymentRepo := payment.NewRepository(r.db)

	// Create availability service for conflict detection
	availabilityRepo := availability.NewRepository(r.db)
	availabilitySvc := availability.NewService(availabilityRepo)

	providerRepo := provider.NewRepository(r.db)
	providerSvc := provider.NewService(providerRepo)

	userRepo := user.NewRepository(r.db)

	emailProvider := notification.NewConsoleEmailProvider()
	notificationSvc := notification.NewService(emailProvider)

	handlers := booking.NewHandlersWithAvailability(bookingRepo, paymentRepo, availabilitySvc, notificationSvc, userRepo, providerSvc)

	api := r.engine.Group("/api")
	bookingsGroup := api.Group("/")
	bookingsGroup.Use(authhandler.Middleware(r.cfg.JWTSecret))
	handlers.Register(bookingsGroup)
}

func (r *Router) registerProviders() {
	repo := provider.NewRepository(r.db)
	service := provider.NewService(repo)
	handlers := provider.NewHandlers(service)

	api := r.engine.Group("/api")
	handlers.Register(api.Group("/"))
}

func (r *Router) registerServices() {
	repo := service.NewRepository(r.db)
	svc := service.NewService(repo)
	handlers := service.NewHandlers(svc)

	api := r.engine.Group("/api")
	handlers.Register(api.Group("/"))
}

func (r *Router) registerFavorites() {
	repo := favorite.NewRepository(r.db)
	svc := favorite.NewService(repo)
	handlers := favorite.NewHandlers(svc)

	api := r.engine.Group("/api")
	favoritesGroup := api.Group("/favorites")
	favoritesGroup.Use(authhandler.Middleware(r.cfg.JWTSecret))
	handlers.Register(favoritesGroup)
}

func (r *Router) registerAvailability() {
	repo := availability.NewRepository(r.db)
	svc := availability.NewService(repo)

	providerRepo := provider.NewRepository(r.db)
	providerSvc := provider.NewService(providerRepo)

	handlers := availability.NewHandlers(svc, providerSvc)

	api := r.engine.Group("/api")
	handlers.Register(api.Group("/"), r.cfg.JWTSecret)
}

func (r *Router) registerReviews() {
	reviewRepo := review.NewRepository(r.db)
	bookingRepo := booking.NewRepository(r.db)
	svc := review.NewService(reviewRepo, bookingRepo)
	handlers := review.NewHandlers(svc)

	api := r.engine.Group("/api")
	handlers.Register(api.Group("/"))

	protected := api.Group("/")
	protected.Use(authhandler.Middleware(r.cfg.JWTSecret))
	handlers.RegisterProtected(protected)
}

func (r *Router) registerPayments() {
	bookingRepo := booking.NewRepository(r.db)
	paymentRepo := payment.NewRepository(r.db)
	handlers := payment.NewHandlersWithFinalizer(paymentRepo, func(paymentID uint, status string) error {
		return bookingRepo.SetStatusByPayment(paymentID, status)
	})

	api := r.engine.Group("/api")
	handlers.Register(api.Group("/"))
}
