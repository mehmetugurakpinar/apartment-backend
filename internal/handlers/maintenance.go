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
	buildingRepo    *repository.BuildingRepository
}

func NewMaintenanceHandler(maintenanceRepo *repository.MaintenanceRepository, buildingRepo ...*repository.BuildingRepository) *MaintenanceHandler {
	h := &MaintenanceHandler{maintenanceRepo: maintenanceRepo}
	if len(buildingRepo) > 0 {
		h.buildingRepo = buildingRepo[0]
	}
	return h
}

func (h *MaintenanceHandler) GetRequests(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var pq models.PaginationQuery
	c.QueryParser(&pq)
	pq.SetDefaults()

	userID := middleware.GetUserID(c)

	// Managers see all requests (including pending_approval).
	// Residents see approved requests + their own pending_approval requests.
	isManager := false
	if h.buildingRepo != nil {
		role, err := h.buildingRepo.GetMemberRole(c.Context(), buildingID, userID)
		if err == nil && (role == models.RoleSuperAdmin || role == models.RoleBuildingManager) {
			isManager = true
		}
	}

	var requests []models.MaintenanceRequestDetail
	var total int64

	if isManager {
		requests, total, err = h.maintenanceRepo.GetByBuilding(c.Context(), buildingID, pq.Page, pq.Limit)
	} else {
		requests, total, err = h.maintenanceRepo.GetByBuildingForResident(c.Context(), buildingID, userID, pq.Page, pq.Limit)
	}
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(models.NewPaginatedResponse(requests, pq.Page, pq.Limit, total), ""))
}

func (h *MaintenanceHandler) ApproveRequest(c *fiber.Ctx) error {
	reqID, err := uuid.Parse(c.Params("reqId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request ID"))
	}

	existing, err := h.maintenanceRepo.GetByID(c.Context(), reqID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse("Request not found"))
	}

	if existing.Status != models.MaintenancePendingApproval {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Only pending requests can be approved"))
	}

	newStatus := models.MaintenanceOpen
	if err := h.maintenanceRepo.Update(c.Context(), reqID, &models.UpdateMaintenanceRequest{
		Status: &newStatus,
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	updated, _ := h.maintenanceRepo.GetByID(c.Context(), reqID)
	return c.JSON(models.SuccessResponse(updated, "Request approved"))
}

func (h *MaintenanceHandler) RejectRequest(c *fiber.Ctx) error {
	reqID, err := uuid.Parse(c.Params("reqId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request ID"))
	}

	existing, err := h.maintenanceRepo.GetByID(c.Context(), reqID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse("Request not found"))
	}

	if existing.Status != models.MaintenancePendingApproval {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Only pending requests can be rejected"))
	}

	newStatus := models.MaintenanceClosed
	if err := h.maintenanceRepo.Update(c.Context(), reqID, &models.UpdateMaintenanceRequest{
		Status: &newStatus,
	}); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	updated, _ := h.maintenanceRepo.GetByID(c.Context(), reqID)
	return c.JSON(models.SuccessResponse(updated, "Request rejected"))
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

	// Determine initial status based on user role:
	// Managers/admins → open (auto-approved)
	// Residents/others → pending_approval (needs manager approval)
	initialStatus := models.MaintenancePendingApproval
	if h.buildingRepo != nil {
		role, err := h.buildingRepo.GetMemberRole(c.Context(), buildingID, userID)
		if err == nil && (role == models.RoleSuperAdmin || role == models.RoleBuildingManager) {
			initialStatus = models.MaintenanceOpen
		}
	}

	mReq := &models.MaintenanceRequest{
		BuildingID:  buildingID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		Status:      initialStatus,
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

	msg := "Maintenance request created"
	if initialStatus == models.MaintenancePendingApproval {
		msg = "Maintenance request submitted for approval"
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(mReq, msg))
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
