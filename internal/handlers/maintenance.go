package handlers

import (
	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MaintenanceHandler struct {
	maintenanceRepo *repository.MaintenanceRepository
}

func NewMaintenanceHandler(maintenanceRepo *repository.MaintenanceRepository) *MaintenanceHandler {
	return &MaintenanceHandler{maintenanceRepo: maintenanceRepo}
}

func (h *MaintenanceHandler) GetRequests(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var pq models.PaginationQuery
	c.QueryParser(&pq)
	pq.SetDefaults()

	requests, total, err := h.maintenanceRepo.GetByBuilding(c.Context(), buildingID, pq.Page, pq.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(models.NewPaginatedResponse(requests, pq.Page, pq.Limit, total), ""))
}

func (h *MaintenanceHandler) CreateRequest(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var req models.CreateMaintenanceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Title is required"))
	}

	userID := middleware.GetUserID(c)
	mReq := &models.MaintenanceRequest{
		BuildingID:  buildingID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		CreatedBy:   userID,
	}

	if req.UnitID != nil {
		unitID, err := uuid.Parse(*req.UnitID)
		if err == nil {
			mReq.UnitID = &unitID
		}
	}

	if err := h.maintenanceRepo.Create(c.Context(), mReq); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(mReq, "Maintenance request created"))
}

func (h *MaintenanceHandler) UpdateRequest(c *fiber.Ctx) error {
	reqID, err := uuid.Parse(c.Params("reqId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request ID"))
	}

	var req models.UpdateMaintenanceRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if err := h.maintenanceRepo.Update(c.Context(), reqID, &req); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	updated, err := h.maintenanceRepo.GetByID(c.Context(), reqID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(updated, "Request updated"))
}

func (h *MaintenanceHandler) CreateVendor(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var req models.CreateVendorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Name == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Vendor name is required"))
	}

	vendor := &models.Vendor{
		BuildingID: buildingID,
		Name:       req.Name,
		Category:   req.Category,
		Phone:      req.Phone,
		Email:      req.Email,
	}

	if err := h.maintenanceRepo.CreateVendor(c.Context(), vendor); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(vendor, "Vendor created"))
}

func (h *MaintenanceHandler) UpdateVendor(c *fiber.Ctx) error {
	vendorID, err := uuid.Parse(c.Params("vendorId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid vendor ID"))
	}

	var req models.UpdateVendorRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	vendor, err := h.maintenanceRepo.UpdateVendor(c.Context(), vendorID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(vendor, "Vendor updated"))
}

func (h *MaintenanceHandler) DeleteVendor(c *fiber.Ctx) error {
	vendorID, err := uuid.Parse(c.Params("vendorId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid vendor ID"))
	}

	if err := h.maintenanceRepo.DeleteVendor(c.Context(), vendorID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Vendor deleted"))
}

func (h *MaintenanceHandler) DeleteRequest(c *fiber.Ctx) error {
	reqID, err := uuid.Parse(c.Params("reqId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request ID"))
	}

	if err := h.maintenanceRepo.DeleteRequest(c.Context(), reqID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Maintenance request deleted"))
}

func (h *MaintenanceHandler) GetVendors(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	vendors, err := h.maintenanceRepo.GetVendors(c.Context(), buildingID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(vendors, ""))
}
