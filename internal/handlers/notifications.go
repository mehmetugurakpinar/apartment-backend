package handlers

import (
	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type NotificationHandler struct {
	notifRepo *repository.NotificationRepository
}

func NewNotificationHandler(notifRepo *repository.NotificationRepository) *NotificationHandler {
	return &NotificationHandler{notifRepo: notifRepo}
}

func (h *NotificationHandler) GetNotifications(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var pq models.PaginationQuery
	c.QueryParser(&pq)
	pq.SetDefaults()

	notifications, total, err := h.notifRepo.GetByUser(c.Context(), userID, pq.Page, pq.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(models.NewPaginatedResponse(notifications, pq.Page, pq.Limit, total), ""))
}

func (h *NotificationHandler) MarkAsRead(c *fiber.Ctx) error {
	notifID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid notification ID"))
	}

	userID := middleware.GetUserID(c)

	if err := h.notifRepo.MarkAsRead(c.Context(), notifID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Notification marked as read"))
}

func (h *NotificationHandler) CreateAnnouncement(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var req models.CreateAnnouncementRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if err := h.notifRepo.CreateBulk(c.Context(), buildingID, req.Type, req.Title, req.Body, nil); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(nil, "Announcement sent to all building members"))
}

func (h *NotificationHandler) GetPreferences(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	prefs, err := h.notifRepo.GetPreferences(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(prefs, ""))
}

func (h *NotificationHandler) UpdatePreferences(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req models.UpdatePreferencesRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if err := h.notifRepo.UpsertPreference(c.Context(), userID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Preferences updated"))
}
