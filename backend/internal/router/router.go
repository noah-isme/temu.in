package router

import (
	"context"
	"errors"
	"log"

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
	"temu.in/backend/internal/models"
	"temu.in/backend/internal/notification"
	"temu.in/backend/internal/payment"
	"temu.in/backend/internal/provider"
	"temu.in/backend/internal/review"
	"temu.in/backend/internal/service"
	"temu.in/backend/internal/token"
	"temu.in/backend/internal/user"
)

type Router struct {
	engine             *gin.Engine
	cfg                *config.Config
	db                 *gorm.DB
	cache              *redis.Client
	notificationSvc    notification.Service
	notificationWorker *notification.Worker
}

func New(engine *gin.Engine, cfg *config.Config, db *gorm.DB, cache *redis.Client) *Router {
	fromEmail := "no-reply@temu.in"
	apiKey := ""
	templateIDs := map[string]string{}
	if cfg != nil {
		if cfg.SendGridFromEmail != "" {
			fromEmail = cfg.SendGridFromEmail
		}
		apiKey = cfg.SendGridAPIKey
		templateIDs = sendGridTemplateIDs(cfg)
	}
	notificationRepo := notification.NewRepository(db)
	notificationSvc := notification.NewService(notificationRepo)
	reminderScheduler, _ := notificationSvc.(notification.ReminderScheduler)
	return &Router{
		engine:             engine,
		cfg:                cfg,
		db:                 db,
		cache:              cache,
		notificationSvc:    notificationSvc,
		notificationWorker: notification.NewWorker(notificationRepo, notification.NewSendGridProvider(apiKey, fromEmail, templateIDs), notification.WorkerConfig{}, reminderScheduler),
	}
}

func sendGridTemplateIDs(cfg *config.Config) map[string]string {
	ids := make(map[string]string)
	for _, template := range []struct {
		name string
		id   string
	}{
		{models.NotificationTypeBookingConfirmationCustomer, cfg.SendGridBookingTemplateID},
		{models.NotificationTypeBookingConfirmationProvider, cfg.SendGridBookingTemplateID},
		{models.NotificationTypeBookingCancellationCustomer, cfg.SendGridCancellationTemplateID},
		{models.NotificationTypeBookingCancellationProvider, cfg.SendGridCancellationTemplateID},
		{models.NotificationTypeBookingReminderCustomer, cfg.SendGridReminderTemplateID},
		{models.NotificationTypeBookingReminderProvider, cfg.SendGridReminderTemplateID},
		{models.NotificationTypePaymentReceipt, cfg.SendGridReceiptTemplateID},
		{models.NotificationTypePasswordReset, cfg.SendGridAuthTemplateID},
		{models.NotificationTypeEmailVerification, cfg.SendGridAuthTemplateID},
	} {
		if template.id != "" {
			ids[template.name] = template.id
		}
	}
	return ids
}

func (r *Router) StartBackgroundWorkers(ctx context.Context) {
	if r.notificationWorker != nil {
		go func() {
			if err := r.notificationWorker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				log.Printf("notification worker stopped: %v", err)
			}
		}()
	}
}

func (r *Router) RegisterAll() {
	r.registerHealth()
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

func (r *Router) registerAuth() {
	repo := user.NewRepository(r.db)
	var tokenRepo *token.Repository
	if r.cache != nil {
		tokenRepo = token.NewRepositoryWithCache(r.db, r.cache)
	} else {
		tokenRepo = token.NewRepository(r.db)
	}

	handler := authhandler.NewHandler(repo, r.cfg, tokenRepo, r.notificationSvc)
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
	paymentProvider := payment.NewMidtransProvider(r.cfg.MidtransServerKey, r.cfg.MidtransClientKey, r.cfg.MidtransBaseURL)
	paymentService := payment.NewService(paymentRepo, paymentProvider)

	// Create availability service for conflict detection
	availabilityRepo := availability.NewRepository(r.db)
	availabilitySvc := availability.NewService(availabilityRepo)

	providerRepo := provider.NewRepository(r.db)
	providerSvc := provider.NewService(providerRepo)
	serviceRepo := service.NewRepository(r.db)
	serviceCatalog := service.NewService(serviceRepo)

	userRepo := user.NewRepository(r.db)

	handlers := booking.NewHandlersWithDependencies(bookingRepo, paymentRepo, availabilitySvc, r.notificationSvc, userRepo, providerSvc, serviceCatalog, paymentService)

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
	handlers.Register(api.Group("/"), r.cfg.JWTSecret)
}

func (r *Router) registerServices() {
	repo := service.NewRepository(r.db)
	svc := service.NewService(repo)
	providerRepo := provider.NewRepository(r.db)
	providerSvc := provider.NewService(providerRepo)
	handlers := service.NewHandlers(svc, providerSvc)

	api := r.engine.Group("/api")
	handlers.Register(api.Group("/"), r.cfg.JWTSecret)
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
	serviceRepo := service.NewRepository(r.db)
	serviceCatalog := service.NewService(serviceRepo)

	handlers := availability.NewHandlers(svc, providerSvc, serviceCatalog)

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
	paymentProvider := payment.NewMidtransProvider(r.cfg.MidtransServerKey, r.cfg.MidtransClientKey, r.cfg.MidtransBaseURL)
	paymentService := payment.NewService(paymentRepo, paymentProvider)
	serviceRepo := service.NewRepository(r.db)
	serviceCatalog := service.NewService(serviceRepo)
	handlers := payment.NewHandlersWithDependencies(paymentRepo, paymentService, serviceCatalog, func(paymentID uint, status string) error {
		if err := bookingRepo.SetStatusByPayment(paymentID, status); err != nil {
			return err
		}
		if status != models.PaymentStatusSucceeded || r.notificationSvc == nil {
			return nil
		}
		receiptService, ok := r.notificationSvc.(notification.PaymentReceiptService)
		if !ok {
			return errors.New("payment receipt notification service is unavailable")
		}

		bookingRecord, err := bookingRepo.FindByPaymentID(paymentID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		paymentRecord, err := paymentRepo.FindByID(paymentID)
		if err != nil {
			return err
		}
		if err := r.notificationSvc.SendBookingConfirmation(context.Background(), bookingRecord, &bookingRecord.User, &bookingRecord.ServiceProvider); err != nil {
			return err
		}
		return receiptService.SendPaymentReceipt(context.Background(), bookingRecord, &bookingRecord.User, paymentRecord)
	})

	api := r.engine.Group("/api")
	protected := api.Group("/")
	protected.Use(authhandler.Middleware(r.cfg.JWTSecret))
	handlers.Register(protected)
	handlers.RegisterWebhook(api.Group("/"))
}
