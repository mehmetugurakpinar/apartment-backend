package handlers

import (
	"apartment-backend/internal/config"
	"apartment-backend/internal/middleware"
	"apartment-backend/internal/repository"
	ws "apartment-backend/internal/websocket"
	"context"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type WSHandler struct {
	hub      *ws.Hub
	cfg      *config.Config
	userRepo *repository.UserRepository
	logger   *zap.Logger
}

func NewWSHandler(hub *ws.Hub, cfg *config.Config, userRepo *repository.UserRepository, logger *zap.Logger) *WSHandler {
	return &WSHandler{hub: hub, cfg: cfg, userRepo: userRepo, logger: logger}
}

func (h *WSHandler) Upgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		// Authenticate via query param
		token := c.Query("token")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing token"})
		}

		// Parse JWT
		claims := &middleware.JWTClaims{}
		_, err := middleware.ParseAccessToken(h.cfg.JWT, token, claims)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}

		c.Locals("userID", claims.UserID)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func (h *WSHandler) Handle() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		userID, ok := c.Locals("userID").(uuid.UUID)
		if !ok {
			c.Close()
			return
		}

		// Get user's buildings
		buildingIDs, err := h.userRepo.GetBuildingsByUser(context.Background(), userID)
		if err != nil || len(buildingIDs) == 0 {
			c.Close()
			return
		}

		// Register a client for each building
		for _, buildingID := range buildingIDs {
			client := &ws.Client{
				ID:         uuid.New(),
				UserID:     userID,
				BuildingID: buildingID,
				Conn:       c,
				Send:       make(chan []byte, 256),
			}

			h.hub.Register(client)

			go ws.WritePump(client)
			// Only read from one goroutine
			if buildingID == buildingIDs[0] {
				ws.ReadPump(h.hub, client)
			}
		}
	})
}
