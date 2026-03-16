package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"apartment-backend/internal/config"
	"apartment-backend/internal/db"
	"apartment-backend/internal/handlers"
	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"
	"apartment-backend/internal/services"
	ws "apartment-backend/internal/websocket"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	// Logger
	logger, _ := zap.NewProduction()
	if os.Getenv("APP_ENV") == "development" {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	// Config
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Database
	pgPool, err := db.NewPostgresPool(cfg.DB, logger)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer pgPool.Close()

	// Redis
	rdb, err := db.NewRedisClient(cfg.Redis, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer rdb.Close()

	// Repositories
	userRepo := repository.NewUserRepository(pgPool)
	buildingRepo := repository.NewBuildingRepository(pgPool)
	financialRepo := repository.NewFinancialRepository(pgPool)
	maintenanceRepo := repository.NewMaintenanceRepository(pgPool)
	notifRepo := repository.NewNotificationRepository(pgPool)
	forumRepo := repository.NewForumRepository(pgPool)
	timelineRepo := repository.NewTimelineRepository(pgPool)
	socialRepo := repository.NewSocialRepository(pgPool)

	// Services
	authService := services.NewAuthService(userRepo, cfg)

	// WebSocket Hub
	hub := ws.NewHub(logger)
	go hub.Run()

	// Handlers
	authHandler := handlers.NewAuthHandler(authService, userRepo)
	buildingHandler := handlers.NewBuildingHandler(buildingRepo, userRepo)
	duesHandler := handlers.NewDuesHandler(financialRepo)
	maintenanceHandler := handlers.NewMaintenanceHandler(maintenanceRepo, buildingRepo)
	notifHandler := handlers.NewNotificationHandler(notifRepo)
	forumHandler := handlers.NewForumHandler(forumRepo, buildingRepo)
	timelineHandler := handlers.NewTimelineHandler(timelineRepo, userRepo, socialRepo)
	socialHandler := handlers.NewSocialHandler(socialRepo)
	_ = handlers.NewWSHandler(hub, cfg, userRepo, logger)

	// Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Apartment Management API",
		ErrorHandler: customErrorHandler,
		BodyLimit:    10 * 1024 * 1024, // 10MB
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.Origins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))
	app.Use(middleware.RequestLogger(logger))
	app.Use(middleware.RateLimiter(rdb, cfg.RateLimit))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "apartment-api"})
	})

	// API v1 routes
	v1 := app.Group("/api/v1")

	// Auth routes (public)
	auth := v1.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.RefreshToken)
	auth.Post("/logout", authHandler.Logout)
	auth.Post("/accept-invitation", buildingHandler.AcceptInvitation)

	// Protected routes
	protected := v1.Group("", middleware.AuthRequired(cfg.JWT))

	// Auth (protected)
	protected.Get("/auth/me", authHandler.Me)
	protected.Patch("/auth/me", authHandler.UpdateProfile)
	protected.Patch("/auth/password", authHandler.ChangePassword)

	// Building-role RBAC middleware: only super_admin or building_manager can proceed
	managerOnly := func(c *fiber.Ctx) error {
		buildingID, err := uuid.Parse(c.Params("id"))
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
		}
		userID := middleware.GetUserID(c)
		role, err := buildingRepo.GetMemberRole(c.Context(), buildingID, userID)
		if err != nil {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse("Not a member of this building"))
		}
		if role != models.RoleSuperAdmin && role != models.RoleBuildingManager {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse("Only admins and managers can perform this action"))
		}
		c.Locals("buildingRole", role)
		return c.Next()
	}

	// Buildings
	buildings := protected.Group("/buildings")
	buildings.Post("/", buildingHandler.Create)
	buildings.Get("/", buildingHandler.GetUserBuildings)
	buildings.Get("/:id", buildingHandler.GetByID)
	buildings.Get("/:id/dashboard", buildingHandler.GetDashboard)
	buildings.Get("/:id/members", buildingHandler.GetMembers)

	// Units (read: all members; write: manager only)
	buildings.Get("/:id/units", buildingHandler.GetUnits)
	buildings.Post("/:id/units", managerOnly, buildingHandler.CreateUnit)
	buildings.Patch("/:id/units/:unitId", managerOnly, buildingHandler.UpdateUnit)
	buildings.Delete("/:id/units/:unitId", managerOnly, buildingHandler.DeleteUnit)

	// Residents
	buildings.Get("/:id/residents", buildingHandler.GetResidents)

	// Members & Invitations (manager only for write)
	buildings.Get("/:id/invitations", buildingHandler.GetInvitations)
	buildings.Post("/:id/invitations", managerOnly, buildingHandler.InviteUser)
	buildings.Delete("/:id/members/:userId", managerOnly, buildingHandler.RemoveMember)

	// Financial (read: all members; write: manager only)
	buildings.Get("/:id/dues", duesHandler.GetDues)
	buildings.Post("/:id/dues", managerOnly, duesHandler.CreateDues)
	buildings.Patch("/:id/dues/:planId", managerOnly, duesHandler.UpdateDues)
	buildings.Delete("/:id/dues/:planId", managerOnly, duesHandler.DeleteDues)
	buildings.Post("/:id/dues/:planId/pay", duesHandler.PayDues)
	buildings.Get("/:id/dues/report", duesHandler.GetReport)
	buildings.Get("/:id/expenses", duesHandler.GetExpenses)
	buildings.Post("/:id/expenses", managerOnly, duesHandler.CreateExpense)
	buildings.Patch("/:id/expenses/:expenseId", managerOnly, duesHandler.UpdateExpense)
	buildings.Delete("/:id/expenses/:expenseId", managerOnly, duesHandler.DeleteExpense)

	// Maintenance (read: all; create: all; approve/update/delete: manager only)
	buildings.Get("/:id/maintenance", maintenanceHandler.GetRequests)
	buildings.Post("/:id/maintenance", maintenanceHandler.CreateRequest)
	buildings.Post("/:id/maintenance/:reqId/approve", managerOnly, maintenanceHandler.ApproveRequest)
	buildings.Post("/:id/maintenance/:reqId/reject", managerOnly, maintenanceHandler.RejectRequest)
	buildings.Patch("/:id/maintenance/:reqId", managerOnly, maintenanceHandler.UpdateRequest)
	buildings.Delete("/:id/maintenance/:reqId", managerOnly, maintenanceHandler.DeleteRequest)

	// Vendors (read: all; write: manager only)
	buildings.Get("/:id/vendors", maintenanceHandler.GetVendors)
	buildings.Post("/:id/vendors", managerOnly, maintenanceHandler.CreateVendor)
	buildings.Patch("/:id/vendors/:vendorId", managerOnly, maintenanceHandler.UpdateVendor)
	buildings.Delete("/:id/vendors/:vendorId", managerOnly, maintenanceHandler.DeleteVendor)

	// Notifications
	protected.Get("/notifications", notifHandler.GetNotifications)
	protected.Patch("/notifications/:id/read", notifHandler.MarkAsRead)
	buildings.Post("/:id/announcements", middleware.RoleRequired(models.RoleSuperAdmin, models.RoleBuildingManager), notifHandler.CreateAnnouncement)
	protected.Get("/notifications/preferences", notifHandler.GetPreferences)
	protected.Patch("/notifications/preferences", notifHandler.UpdatePreferences)

	// Users / Social (search MUST be before :id routes)
	protected.Get("/users/search", socialHandler.SearchUsers)
	protected.Post("/users/:id/follow", socialHandler.FollowUser)
	protected.Delete("/users/:id/follow", socialHandler.UnfollowUser)
	protected.Get("/users/:id/followers", socialHandler.GetFollowers)
	protected.Get("/users/:id/following", socialHandler.GetFollowing)

	// Forum
	buildings.Get("/:id/forum/categories", forumHandler.GetCategories)
	buildings.Get("/:id/forum/posts", forumHandler.GetPosts)
	buildings.Post("/:id/forum/posts", forumHandler.CreatePost)
	buildings.Get("/:id/forum/posts/:postId", forumHandler.GetPost)
	buildings.Post("/:id/forum/posts/:postId/comments", forumHandler.CreateComment)
	buildings.Post("/:id/forum/posts/:postId/vote", forumHandler.Vote)

	// Timeline
	protected.Get("/timeline", timelineHandler.GetFeed)
	protected.Post("/timeline", timelineHandler.CreatePost)
	protected.Post("/timeline/:postId/like", timelineHandler.LikePost)
	protected.Post("/timeline/:postId/comments", timelineHandler.CreateComment)
	protected.Post("/timeline/:postId/repost", timelineHandler.RepostPost)
	protected.Delete("/timeline/:postId/repost", timelineHandler.UnrepostPost)
	protected.Post("/timeline/polls/:pollId/vote", timelineHandler.VotePoll)
	protected.Get("/timeline/nearby", timelineHandler.GetNearby)

	// WebSocket (handled with query param auth)
	// app.Use("/ws", wsHandler.Upgrade)
	// app.Get("/ws", wsHandler.Handle())

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := fmt.Sprintf("%s:%s", cfg.App.Host, cfg.App.Port)
		logger.Info("Server starting", zap.String("addr", addr))
		if err := app.Listen(addr); err != nil {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("Shutting down server...")
	app.Shutdown()
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(models.ErrorResponse(err.Error()))
}
