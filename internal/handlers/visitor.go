package handlers

import (
	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type VisitorHandler struct {
	visitorRepo *repository.VisitorRepository
}

func NewVisitorHandler(visitorRepo *repository.VisitorRepository) *VisitorHandler {
	return &VisitorHandler{visitorRepo: visitorRepo}
}

func (h *VisitorHandler) CreatePass(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}
	userID := middleware.GetUserID(c)

	var req models.CreateVisitorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}
	if req.VisitorName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Visitor name is required"))
	}

	pass, err := h.visitorRepo.Create(c.Context(), buildingID, userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(pass, "Visitor pass created"))
}

func (h *VisitorHandler) GetPasses(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	status := c.Query("status", "all")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	passes, err := h.visitorRepo.GetByBuilding(c.Context(), buildingID, status, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(passes, ""))
}

func (h *VisitorHandler) CheckIn(c *fiber.Ctx) error {
	passID, err := uuid.Parse(c.Params("passId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid pass ID"))
	}
	userID := middleware.GetUserID(c)

	if err := h.visitorRepo.CheckIn(c.Context(), passID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(nil, "Visitor checked in"))
}

func (h *VisitorHandler) CheckOut(c *fiber.Ctx) error {
	passID, err := uuid.Parse(c.Params("passId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid pass ID"))
	}

	if err := h.visitorRepo.CheckOut(c.Context(), passID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(nil, "Visitor checked out"))
}

func (h *VisitorHandler) CancelPass(c *fiber.Ctx) error {
	passID, err := uuid.Parse(c.Params("passId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid pass ID"))
	}

	if err := h.visitorRepo.Cancel(c.Context(), passID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(nil, "Visitor pass cancelled"))
}

func (h *VisitorHandler) ScanQR(c *fiber.Ctx) error {
	qrCode := c.Params("qr")
	if qrCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("QR code is required"))
	}

	pass, err := h.visitorRepo.GetByQR(c.Context(), qrCode)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse("Visitor pass not found"))
	}
	return c.JSON(models.SuccessResponse(pass, ""))
}
