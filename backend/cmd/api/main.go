package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hirescope/backend/internal/calendar"
	"hirescope/backend/internal/config"
	"hirescope/backend/internal/crypto"
	"hirescope/backend/internal/database"
	"hirescope/backend/internal/email"
	"hirescope/backend/internal/extractor"
	"hirescope/backend/internal/handler"
	"hirescope/backend/internal/middleware"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"
	"hirescope/backend/internal/storage"
	"hirescope/backend/internal/worker"
	"hirescope/backend/migrations"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Fatal: Configuration error: %v", err)
	}

	// Set Gin operating mode
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 2. Initialize database connection
	dbWrapper, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Fatal: Database connection error: %v", err)
	}
	defer func() {
		if err := dbWrapper.Close(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}()
	log.Printf("Successfully connected to PostgreSQL database '%s' at %s:%s", cfg.DBName, cfg.DBHost, cfg.DBPort)

	// 3. Run safe non-destructive migrations
	if err := migrations.RunMigrations(dbWrapper.DB); err != nil {
		log.Fatalf("Fatal: Database migration failure: %v", err)
	}
	log.Println("Database migrations executed successfully.")

	// 4. Seed initial development users and sample jobs if in development mode
	if cfg.AppEnv == "development" {
		if err := database.SeedDevelopmentUsers(dbWrapper.DB); err != nil {
			log.Printf("Warning: Failed to seed development users: %v", err)
		}
		if err := database.SeedDevelopmentJobs(dbWrapper.DB); err != nil {
			log.Printf("Warning: Failed to seed development jobs: %v", err)
		}
	}

	// 5. Initialize Services and Repositories
	jwtService, err := service.NewJWTService(cfg.JWTSecret, cfg.JWTExpirationHours)
	if err != nil {
		log.Fatalf("Fatal: JWT service initialization error: %v", err)
	}

	userRepo := repository.NewUserRepository(dbWrapper.DB)
	revokedTokenRepo := repository.NewRevokedTokenRepository(dbWrapper.DB)
	jobRepo := repository.NewJobRepository(dbWrapper.DB)
	candidateRepo := repository.NewCandidateRepository(dbWrapper.DB)
	screeningRepo := repository.NewScreeningRepository(dbWrapper.DB)
	jobCandidateRepo := repository.NewJobCandidateRepository(dbWrapper.DB)
	candidateNoteRepo := repository.NewCandidateNoteRepository(dbWrapper.DB)
	auditLogRepo := repository.NewAuditLogRepository(dbWrapper.DB)

	storageService, err := storage.NewLocalStorageService(cfg.CVStorageDir)
	if err != nil {
		log.Fatalf("Fatal: Storage service initialization error: %v", err)
	}
	pdfExtractor := extractor.NewPDFTextExtractor()
	docxExtractor := extractor.NewDOCXTextExtractor()

	authService := service.NewAuthService(userRepo, revokedTokenRepo, jwtService)
	jobService := service.NewJobService(jobRepo)
	candidateService := service.NewCandidateService(candidateRepo, jobRepo, storageService, pdfExtractor, docxExtractor)
	screeningEngine := service.NewScreeningEngine()
	screeningService := service.NewScreeningService(screeningRepo, jobRepo, candidateRepo, screeningEngine)
	reviewService := service.NewCandidateReviewService(jobCandidateRepo, candidateNoteRepo, jobRepo, candidateRepo, screeningRepo, auditLogRepo)
	dashboardRepo := repository.NewDashboardRepository(dbWrapper.DB)
	dashboardService := service.NewDashboardService(dashboardRepo)

	// Step 12B-2: Email Provider and Notification Setup
	var emailProvider email.EmailProvider
	if cfg.EmailEnabled && cfg.EmailProvider == "resend" {
		emailProvider = email.NewResendEmailProvider(cfg.ResendAPIKey)
		log.Println("Email delivery provider initialized: Resend (ENABLED)")
	} else {
		emailProvider = email.NewDisabledEmailProvider()
		log.Println("Email delivery provider initialized: Disabled (EMAIL_ENABLED=false)")
	}

	emailDeliveryRepo := repository.NewInterviewEmailDeliveryRepository(dbWrapper.DB)
	interviewRepo := repository.NewInterviewRepository(dbWrapper.DB)
	notificationService := service.NewInterviewNotificationService(
		emailDeliveryRepo,
		interviewRepo,
		userRepo,
		auditLogRepo,
		emailProvider,
		cfg,
	)

	interviewService := service.NewInterviewService(interviewRepo, jobCandidateRepo, userRepo, notificationService)

	// External Calendar Integration (Step 12B-3)
	tokenEncryptor, err := crypto.NewTokenEncryptor(cfg.CalendarTokenEncryptionKey)
	if err != nil {
		log.Fatalf("Fatal: Failed to initialize token encryptor: %v", err)
	}
	calendarStateManager := calendar.NewStateManager(10 * time.Minute)
	googleProvider := calendar.NewGoogleCalendarProvider(calendar.GoogleCalendarConfig{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
	}, nil)
	microsoftProvider := calendar.NewMicrosoftCalendarProvider(calendar.MicrosoftCalendarConfig{
		ClientID:     cfg.MicrosoftClientID,
		ClientSecret: cfg.MicrosoftClientSecret,
		TenantID:     cfg.MicrosoftTenantID,
		RedirectURL:  cfg.MicrosoftRedirectURL,
	}, nil)

	calendarRepo := repository.NewCalendarRepository(dbWrapper.DB)
	calendarService := service.NewCalendarIntegrationService(
		calendarRepo,
		interviewRepo,
		userRepo,
		auditLogRepo,
		tokenEncryptor,
		calendarStateManager,
		googleProvider,
		microsoftProvider,
		cfg,
	)
	interviewService.SetCalendarIntegrationService(calendarService)

	// Background reminder worker
	reminderWorker := worker.NewReminderWorker(notificationService, 1*time.Minute, cfg.EmailEnabled)
	reminderWorker.Start()
	defer reminderWorker.Stop()

	// 6. Setup Handlers
	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService)
	jobHandler := handler.NewJobHandler(jobService)
	candidateHandler := handler.NewCandidateHandler(candidateService)
	screeningHandler := handler.NewScreeningHandler(screeningService)
	reviewHandler := handler.NewCandidateReviewHandler(reviewService)
	dashboardHandler := handler.NewDashboardHandler(dashboardService)
	interviewHandler := handler.NewInterviewHandler(interviewService, notificationService)
	calendarHandler := handler.NewCalendarHandler(calendarService, cfg)
	analyticsRepo := repository.NewAnalyticsRepository(dbWrapper.DB)
	analyticsService := service.NewAnalyticsService(analyticsRepo, auditLogRepo)
	analyticsHandler := handler.NewAnalyticsHandler(analyticsService)

	// Rate limiters for sensitive endpoints (Step 14)
	loginLimiter := middleware.NewIPRateLimiter(20, 1*time.Minute)

	// 7. Setup Gin HTTP router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(middleware.SecurityHeaders())

	// Setup CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:5173"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "Accept"}
	router.Use(cors.New(corsConfig))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health check
		v1.GET("/health", healthHandler.Check)

		// Public authentication routes
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/login", loginLimiter.Limit(), authHandler.Login)

			// Protected authentication routes
			protectedAuth := authGroup.Group("")
			protectedAuth.Use(middleware.AuthMiddleware(jwtService, revokedTokenRepo))
			{
				protectedAuth.POST("/logout", authHandler.Logout)
				protectedAuth.GET("/me", authHandler.Me)
			}
		}

		// Protected job vacancy & requirements routes
		jobsGroup := v1.Group("/jobs")
		jobsGroup.Use(middleware.AuthMiddleware(jwtService, revokedTokenRepo))
		{
			jobsGroup.POST("", jobHandler.Create)
			jobsGroup.GET("", jobHandler.List)
			jobsGroup.GET("/:id", jobHandler.GetByID)
			jobsGroup.PUT("/:id", jobHandler.Update)
			jobsGroup.PATCH("/:id/status", jobHandler.UpdateStatus)
			jobsGroup.POST("/:id/archive", jobHandler.Archive)

			// Nested requirement routes
			jobsGroup.GET("/:id/requirements", jobHandler.ListRequirements)
			jobsGroup.POST("/:id/requirements", jobHandler.CreateRequirement)
			jobsGroup.PUT("/:id/requirements/:requirementId", jobHandler.UpdateRequirement)
			jobsGroup.DELETE("/:id/requirements/:requirementId", jobHandler.DeleteRequirement)

			// Job-Candidate association & recruiter review routes
			jobsGroup.POST("/:id/candidates", candidateHandler.AddJobCandidate)
			jobsGroup.GET("/:id/candidates", reviewHandler.ListCandidatesByJob)
			jobsGroup.DELETE("/:id/candidates/:candidateId", candidateHandler.RemoveJobCandidate)

			// Candidate workflow & recruiter review routes (Step 8)
			jobsGroup.GET("/:id/candidates/:candidateId/review", reviewHandler.GetReviewDetail)
			jobsGroup.PATCH("/:id/candidates/:candidateId/status", reviewHandler.UpdateStatus)
			jobsGroup.POST("/:id/candidates/:candidateId/notes", reviewHandler.CreateNote)
			jobsGroup.GET("/:id/candidates/:candidateId/notes", reviewHandler.ListNotes)
			jobsGroup.PUT("/:id/candidates/:candidateId/notes/:noteId", reviewHandler.UpdateNote)
			jobsGroup.DELETE("/:id/candidates/:candidateId/notes/:noteId", reviewHandler.DeleteNote)

			// Candidate screening routes (Step 7)
			jobsGroup.POST("/:id/candidates/:candidateId/screen", screeningHandler.ScreenCandidate)
			jobsGroup.GET("/:id/candidates/:candidateId/screening", screeningHandler.GetLatestScreening)
		}

		// Protected candidate management routes
		candidatesGroup := v1.Group("/candidates")
		candidatesGroup.Use(middleware.AuthMiddleware(jwtService, revokedTokenRepo))
		{
			candidatesGroup.POST("", candidateHandler.Create)
			candidatesGroup.GET("", candidateHandler.List)
			candidatesGroup.GET("/:id", candidateHandler.GetByID)
			candidatesGroup.PUT("/:id", candidateHandler.Update)

			// Associated jobs for candidate
			candidatesGroup.GET("/:id/jobs", candidateHandler.ListJobsByCandidate)

			// Nested education routes
			candidatesGroup.POST("/:id/educations", candidateHandler.CreateEducation)
			candidatesGroup.GET("/:id/educations", candidateHandler.ListEducations)
			candidatesGroup.PUT("/:id/educations/:educationId", candidateHandler.UpdateEducation)
			candidatesGroup.DELETE("/:id/educations/:educationId", candidateHandler.DeleteEducation)

			// Nested experience routes
			candidatesGroup.POST("/:id/experiences", candidateHandler.CreateExperience)
			candidatesGroup.GET("/:id/experiences", candidateHandler.ListExperiences)
			candidatesGroup.PUT("/:id/experiences/:experienceId", candidateHandler.UpdateExperience)
			candidatesGroup.DELETE("/:id/experiences/:experienceId", candidateHandler.DeleteExperience)

			// Nested skill routes
			candidatesGroup.POST("/:id/skills", candidateHandler.CreateSkill)
			candidatesGroup.GET("/:id/skills", candidateHandler.ListSkills)
			candidatesGroup.PUT("/:id/skills/:skillId", candidateHandler.UpdateSkill)
			candidatesGroup.DELETE("/:id/skills/:skillId", candidateHandler.DeleteSkill)

			// Nested document routes (TEXT, PDF, and DOCX processing)
			candidatesGroup.POST("/:id/documents", candidateHandler.CreateDocument)
			candidatesGroup.POST("/:id/documents/upload", candidateHandler.UploadDocument)
			candidatesGroup.POST("/:id/documents/:documentId/process", candidateHandler.ProcessDocument)
			candidatesGroup.GET("/:id/documents", candidateHandler.ListDocuments)
			candidatesGroup.GET("/:id/documents/:documentId", candidateHandler.GetDocumentByID)
			candidatesGroup.DELETE("/:id/documents/:documentId", candidateHandler.DeleteDocument)

			// Nested candidate interview routes (Step 12A)
			candidatesGroup.GET("/:id/interviews", interviewHandler.GetCandidateInterviews)
			candidatesGroup.GET("/:id/next-interview", interviewHandler.GetCandidateNextInterview)
		}

		// Protected dashboard intelligence routes (Step 11)
		dashboardGroup := v1.Group("/dashboard")
		dashboardGroup.Use(middleware.AuthMiddleware(jwtService, revokedTokenRepo))
		{
			dashboardGroup.GET("/summary", dashboardHandler.GetSummary)
			dashboardGroup.GET("/recent-candidates", dashboardHandler.GetRecentCandidates)
			dashboardGroup.GET("/open-jobs", dashboardHandler.GetOpenJobs)
			dashboardGroup.GET("/activity", dashboardHandler.GetActivity)
		}

		// Protected interview management routes (Step 12A)
		interviewsGroup := v1.Group("/interviews")
		interviewsGroup.Use(middleware.AuthMiddleware(jwtService, revokedTokenRepo))
		{
			interviewsGroup.POST("", interviewHandler.CreateInterview)
			interviewsGroup.GET("", interviewHandler.ListInterviews)
			interviewsGroup.GET("/:id", interviewHandler.GetInterview)
			interviewsGroup.GET("/:id/ics", interviewHandler.ExportICS)
			interviewsGroup.PUT("/:id", interviewHandler.UpdateInterview)
			interviewsGroup.PATCH("/:id/reschedule", interviewHandler.RescheduleInterview)
			interviewsGroup.PATCH("/:id/cancel", interviewHandler.CancelInterview)
			interviewsGroup.POST("/:id/complete", interviewHandler.CompleteInterview)

			// Email notifications & deliveries (Step 12B-2)
			interviewsGroup.GET("/:id/email-deliveries", interviewHandler.ListEmailDeliveries)
			interviewsGroup.POST("/:id/send-invitation", interviewHandler.SendInvitation)
			interviewsGroup.POST("/:id/email-deliveries/:deliveryId/retry", interviewHandler.RetryEmailDelivery)

			// External Calendar Integration (Step 12B-3)
			interviewsGroup.GET("/:id/calendar/events", calendarHandler.GetInterviewCalendarEvents)
			interviewsGroup.POST("/:id/calendar/sync/:provider", calendarHandler.SyncInterview)
			interviewsGroup.POST("/:id/calendar/retry/:provider", calendarHandler.RetrySync)
		}

		// Calendar integrations routes (Step 12B-3)
		calendarGroup := v1.Group("/calendar")
		{
			// Public OAuth callback endpoints (OAuth provider browser redirect does not supply JWT)
			calendarGroup.GET("/google/callback", calendarHandler.CallbackGoogle)
			calendarGroup.GET("/microsoft/callback", calendarHandler.CallbackMicrosoft)

			// Protected connection management endpoints
			calendarAuth := calendarGroup.Group("")
			calendarAuth.Use(middleware.AuthMiddleware(jwtService, revokedTokenRepo))
			calendarAuth.Use(middleware.RequireRole(model.RoleAdmin, model.RoleRecruiter))
			{
				calendarAuth.GET("/connections", calendarHandler.GetConnections)
				calendarAuth.GET("/google/connect", calendarHandler.ConnectGoogle)
				calendarAuth.DELETE("/google", calendarHandler.DisconnectGoogle)
				calendarAuth.GET("/microsoft/connect", calendarHandler.ConnectMicrosoft)
				calendarAuth.DELETE("/microsoft", calendarHandler.DisconnectMicrosoft)
			}
		}

		// Protected users routes (for interviewer selection)
		usersGroup := v1.Group("/users")
		usersGroup.Use(middleware.AuthMiddleware(jwtService, revokedTokenRepo))
		{
			usersGroup.GET("", interviewHandler.ListUsers)
		}

		// Protected recruitment analytics & reporting routes (Step 13)
		analyticsGroup := v1.Group("/analytics")
		analyticsGroup.Use(middleware.AuthMiddleware(jwtService, revokedTokenRepo))
		analyticsGroup.Use(middleware.RequireRole(model.RoleAdmin, model.RoleRecruiter))
		{
			analyticsGroup.GET("/overview", analyticsHandler.GetOverview)
			analyticsGroup.GET("/jobs", analyticsHandler.GetJobPerformance)
			analyticsGroup.GET("/recruiters", analyticsHandler.GetRecruiterActivity)
			analyticsGroup.GET("/departments", analyticsHandler.GetDepartments)
			analyticsGroup.GET("/export", analyticsHandler.ExportCSV)
		}
	}

	// 8. Configure HTTP server
	addr := ":" + cfg.AppPort
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// 9. Run server in a background goroutine
	go func() {
		log.Printf("Server listening on %s (env: %s)", addr, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Fatal: HTTP server failure: %v", err)
		}
	}()

	// 10. Graceful shutdown on SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal '%s'. Initiating graceful shutdown...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting gracefully.")
}
