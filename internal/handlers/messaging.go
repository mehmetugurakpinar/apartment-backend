package handlers

import (
	"time"

	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"
	ws "apartment-backend/internal/websocket"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MessagingHandler struct {
	messagingRepo *repository.MessagingRepository
	hub           *ws.Hub
}

func NewMessagingHandler(messagingRepo *repository.MessagingRepository, hub *ws.Hub) *MessagingHandler {
	return &MessagingHandler{messagingRepo: messagingRepo, hub: hub}
}

// StartConversation creates or retrieves a 1:1 conversation with another user.
func (h *MessagingHandler) StartConversation(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req models.CreateConversationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	otherUserID, err := uuid.Parse(req.UserID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid user_id"))
	}

	if userID == otherUserID {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Cannot start conversation with yourself"))
	}

	conv, err := h.messagingRepo.GetOrCreateDirectConversation(c.Context(), userID, otherUserID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(conv, "Conversation ready"))
}

// GetConversations returns the user's conversations with last message preview.
func (h *MessagingHandler) GetConversations(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var pq models.PaginationQuery
	c.QueryParser(&pq)
	pq.SetDefaults()

	conversations, total, err := h.messagingRepo.GetConversations(c.Context(), userID, pq.Page, pq.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(models.NewPaginatedResponse(conversations, pq.Page, pq.Limit, total), ""))
}

// GetMessages returns messages for a conversation with cursor-based pagination.
func (h *MessagingHandler) GetMessages(c *fiber.Ctx) error {
	convID, err := uuid.Parse(c.Params("convId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid conversation ID"))
	}

	userID := middleware.GetUserID(c)

	// Authorization: check user is participant
	isParticipant, err := h.messagingRepo.IsParticipant(c.Context(), convID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	if !isParticipant {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse("Not a participant of this conversation"))
	}

	// Parse cursor
	var before *time.Time
	if b := c.Query("before"); b != "" {
		t, err := time.Parse(time.RFC3339Nano, b)
		if err == nil {
			before = &t
		}
	}

	limit := c.QueryInt("limit", 50)
	if limit < 1 || limit > 100 {
		limit = 50
	}

	messages, err := h.messagingRepo.GetMessages(c.Context(), convID, before, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(messages, ""))
}

// SendMessage sends a message to a conversation.
func (h *MessagingHandler) SendMessage(c *fiber.Ctx) error {
	convID, err := uuid.Parse(c.Params("convId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid conversation ID"))
	}

	userID := middleware.GetUserID(c)

	// Authorization
	isParticipant, err := h.messagingRepo.IsParticipant(c.Context(), convID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	if !isParticipant {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse("Not a participant of this conversation"))
	}

	var req models.SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Content is required"))
	}

	if req.MessageType == "" {
		req.MessageType = models.MessageTypeText
	}

	msg, err := h.messagingRepo.SendMessage(c.Context(), convID, userID, req.Content, req.MessageType)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	// Send real-time notification via WebSocket to other participants
	if h.hub != nil {
		participantIDs, _ := h.messagingRepo.GetParticipantIDs(c.Context(), convID)
		for _, pid := range participantIDs {
			if pid != userID {
				h.hub.SendToUser(pid, ws.EventNewMessage, msg)
			}
		}
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(msg, "Message sent"))
}

// MarkAsRead marks all messages in a conversation as read for the user.
func (h *MessagingHandler) MarkAsRead(c *fiber.Ctx) error {
	convID, err := uuid.Parse(c.Params("convId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid conversation ID"))
	}

	userID := middleware.GetUserID(c)

	// Authorization
	isParticipant, err := h.messagingRepo.IsParticipant(c.Context(), convID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	if !isParticipant {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse("Not a participant of this conversation"))
	}

	if err := h.messagingRepo.MarkAsRead(c.Context(), convID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Marked as read"))
}
