package handlers

import (
	"time"

	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type BuildingHandler struct {
	buildingRepo *repository.BuildingRepository
	userRepo     *repository.UserRepository
}

func NewBuildingHandler(buildingRepo *repository.BuildingRepository, userRepo ...*repository.UserRepository) *BuildingHandler {
	h := &BuildingHandler{buildingRepo: buildingRepo}
	if len(userRepo) > 0 {
		h.userRepo = userRepo[0]
	}
	return h
}

func (h *BuildingHandler) Create(c *fiber.Ctx) error {
	var req models.CreateBuildingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Name == "" || req.Address == "" || req.City == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Name, address, and city are required"))
	}

	userID := middleware.GetUserID(c)
	building := &models.Building{
		Name:       req.Name,
		Address:    req.Address,
		City:       req.City,
		TotalUnits: req.TotalUnits,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		CreatedBy:  &userID,
	}

	if err := h.buildingRepo.Create(c.Context(), building); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	// Add creator as building manager
	member := &models.BuildingMember{
		BuildingID: building.ID,
		UserID:     userID,
		Role:       models.RoleBuildingManager,
	}
	h.buildingRepo.AddMember(c.Context(), member)

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(building, "Building created"))
}

func (h *BuildingHandler) GetUserBuildings(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	buildings, err := h.buildingRepo.GetByUserID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}
	if buildings == nil {
		buildings = []models.Building{}
	}
	return c.JSON(models.SuccessResponse(buildings, ""))
}

func (h *BuildingHandler) GetByID(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	userID := middleware.GetUserID(c)
	isMember, _ := h.buildingRepo.IsMember(c.Context(), id, userID)
	if !isMember && middleware.GetUserRole(c) != models.RoleSuperAdmin {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse("Access denied"))
	}

	building, err := h.buildingRepo.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(building, ""))
}

func (h *BuildingHandler) GetDashboard(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	userID := middleware.GetUserID(c)
	isMember, _ := h.buildingRepo.IsMember(c.Context(), id, userID)
	if !isMember && middleware.GetUserRole(c) != models.RoleSuperAdmin {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse("Access denied"))
	}

	dashboard, err := h.buildingRepo.GetDashboard(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(dashboard, ""))
}

func (h *BuildingHandler) GetUnits(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	units, err := h.buildingRepo.GetUnits(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(units, ""))
}

func (h *BuildingHandler) CreateUnit(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var req models.CreateUnitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.UnitNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Unit number is required"))
	}

	unit := &models.Unit{
		BuildingID: buildingID,
		Block:      req.Block,
		Floor:      req.Floor,
		UnitNumber: req.UnitNumber,
		AreaSqm:    req.AreaSqm,
		Status:     models.UnitVacant,
	}

	if req.OwnerID != nil {
		ownerID, err := uuid.Parse(*req.OwnerID)
		if err == nil {
			unit.OwnerID = &ownerID
			unit.Status = models.UnitOccupied
		}
	}

	if err := h.buildingRepo.CreateUnit(c.Context(), unit); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(unit, "Unit created"))
}

func (h *BuildingHandler) UpdateUnit(c *fiber.Ctx) error {
	unitID, err := uuid.Parse(c.Params("unitId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid unit ID"))
	}

	var req models.UpdateUnitRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	unit, err := h.buildingRepo.UpdateUnit(c.Context(), unitID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(unit, "Unit updated"))
}

func (h *BuildingHandler) GetResidents(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	residents, err := h.buildingRepo.GetResidents(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(residents, ""))
}

func (h *BuildingHandler) DeleteUnit(c *fiber.Ctx) error {
	unitID, err := uuid.Parse(c.Params("unitId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid unit ID"))
	}

	if err := h.buildingRepo.DeleteUnit(c.Context(), unitID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Unit deleted"))
}

func (h *BuildingHandler) GetMembers(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	members, err := h.buildingRepo.GetMembers(c.Context(), buildingID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(members, ""))
}

func (h *BuildingHandler) RemoveMember(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid user ID"))
	}

	// Prevent removing yourself
	currentUserID := middleware.GetUserID(c)
	if currentUserID == userID {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Cannot remove yourself from the building"))
	}

	if err := h.buildingRepo.RemoveMember(c.Context(), buildingID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Member removed"))
}

// Invitations

func (h *BuildingHandler) InviteUser(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	var req models.CreateInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Email is required"))
	}

	if req.Role == "" {
		req.Role = models.RoleResident
	}

	// Enforce single manager per building
	if req.Role == models.RoleBuildingManager {
		hasManager, _ := h.buildingRepo.HasManager(c.Context(), buildingID)
		if hasManager {
			return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse("This building already has a manager. A building can only have one manager."))
		}
	}

	userID := middleware.GetUserID(c)

	invitation := &models.BuildingInvitation{
		BuildingID: buildingID,
		Email:      req.Email,
		Role:       req.Role,
		Token:      uuid.New().String(),
		InvitedBy:  userID,
		ExpiresAt:  time.Now().Add(7 * 24 * time.Hour),
	}

	if err := h.buildingRepo.CreateInvitation(c.Context(), invitation); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(invitation, "Invitation created"))
}

func (h *BuildingHandler) GetInvitations(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}

	invitations, err := h.buildingRepo.GetInvitationsByBuilding(c.Context(), buildingID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(invitations, ""))
}

func (h *BuildingHandler) AcceptInvitation(c *fiber.Ctx) error {
	var req models.AcceptInvitationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Token == "" || req.Password == "" || req.FullName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Token, password, and full_name are required"))
	}

	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Password must be at least 8 characters"))
	}

	// Get invitation by token
	invitation, err := h.buildingRepo.GetInvitationByToken(c.Context(), req.Token)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse("Invalid invitation token"))
	}

	// Check if already accepted
	if invitation.AcceptedAt != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invitation has already been accepted"))
	}

	// Check if expired
	if time.Now().After(invitation.ExpiresAt) {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invitation has expired"))
	}

	// Check if user already exists
	existingUser, err := h.userRepo.GetByEmail(c.Context(), invitation.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	var user *models.User
	if existingUser != nil {
		user = existingUser
	} else {
		// Create new user
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse("Failed to hash password"))
		}

		user = &models.User{
			Email:        invitation.Email,
			PasswordHash: string(hash),
			FullName:     req.FullName,
			Role:         invitation.Role,
			IsActive:     true,
		}

		if err := h.userRepo.Create(c.Context(), user); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse("Failed to create user: " + err.Error()))
		}
	}

	// Add user to building members
	member := &models.BuildingMember{
		BuildingID: invitation.BuildingID,
		UserID:     user.ID,
		Role:       invitation.Role,
	}
	if err := h.buildingRepo.AddMember(c.Context(), member); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse("Failed to add user to building: " + err.Error()))
	}

	// Mark invitation as accepted
	if err := h.buildingRepo.MarkInvitationAccepted(c.Context(), invitation.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse("Failed to mark invitation as accepted"))
	}

	return c.JSON(models.SuccessResponse(user.ToResponse(), "Invitation accepted successfully"))
}
