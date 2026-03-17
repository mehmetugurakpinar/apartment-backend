package handlers

import (
	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ReservationHandler struct {
	reservationRepo *repository.ReservationRepository
}

func NewReservationHandler(reservationRepo *repository.ReservationRepository) *ReservationHandler {
	return &ReservationHandler{reservationRepo: reservationRepo}
}

// Common Areas

func (h *ReservationHandler) CreateArea(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var req models.CreateCommonAreaRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}
	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Name is required"))
	}

	area, err := h.reservationRepo.CreateArea(c.Context(), buildingID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(area, "Common area created"))
}

func (h *ReservationHandler) GetAreas(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	areas, err := h.reservationRepo.GetAreas(c.Context(), buildingID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(areas, ""))
}

// Reservations

func (h *ReservationHandler) CreateReservation(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}
	userID := middleware.GetUserID(c)

	var req models.CreateReservationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}
	if req.CommonAreaID == "" || req.StartTime == "" || req.EndTime == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Area, start time, and end time are required"))
	}

	reservation, err := h.reservationRepo.CreateReservation(c.Context(), buildingID, userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(reservation, "Reservation created"))
}

func (h *ReservationHandler) GetReservations(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	var areaID *uuid.UUID
	if areaStr := c.Query("area_id"); areaStr != "" {
		id, err := uuid.Parse(areaStr)
		if err == nil {
			areaID = &id
		}
	}

	reservations, err := h.reservationRepo.GetReservations(c.Context(), buildingID, areaID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(reservations, ""))
}

func (h *ReservationHandler) ApproveReservation(c *fiber.Ctx) error {
	resID, err := uuid.Parse(c.Params("resId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid reservation ID"))
	}

	if err := h.reservationRepo.UpdateReservationStatus(c.Context(), resID, "approved"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(nil, "Reservation approved"))
}

func (h *ReservationHandler) RejectReservation(c *fiber.Ctx) error {
	resID, err := uuid.Parse(c.Params("resId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid reservation ID"))
	}

	if err := h.reservationRepo.UpdateReservationStatus(c.Context(), resID, "rejected"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(nil, "Reservation rejected"))
}

func (h *ReservationHandler) CancelReservation(c *fiber.Ctx) error {
	resID, err := uuid.Parse(c.Params("resId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid reservation ID"))
	}

	if err := h.reservationRepo.UpdateReservationStatus(c.Context(), resID, "cancelled"); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(nil, "Reservation cancelled"))
}

func (h *ReservationHandler) GetMyReservations(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	reservations, err := h.reservationRepo.GetMyReservations(c.Context(), userID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	return c.JSON(models.SuccessResponse(reservations, ""))
}
