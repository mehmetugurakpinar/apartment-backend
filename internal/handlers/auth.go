package handlers

import (
	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"
	"apartment-backend/internal/services"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authService *services.AuthService
	userRepo    *repository.UserRepository
}

func NewAuthHandler(authService *services.AuthService, userRepo *repository.UserRepository) *AuthHandler {
	return &AuthHandler{authService: authService, userRepo: userRepo}
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Email == "" || req.Password == "" || req.FullName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Email, password, and full_name are required"))
	}

	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Password must be at least 8 characters"))
	}

	resp, err := h.authService.Register(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(resp, "Registration successful"))
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Email and password are required"))
	}

	resp, err := h.authService.Login(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(resp, "Login successful"))
}

func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req models.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	resp, err := h.authService.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(resp, "Token refreshed"))
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req models.RefreshRequest
	log.Info("HTTP request",
		zap.String("method", req.RefreshToken),
	)
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if err := h.authService.Logout(c.Context(), req.RefreshToken); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Logged out successfully"))
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(user.ToResponse(), ""))
}

func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)
	log.Info("HTTP request",
		zap.String("method", userID.String()),
	)
	var req models.UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	// Prevent self-assigning super_admin
	if req.Role != nil && *req.Role == models.RoleSuperAdmin {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse("Cannot self-assign super_admin role"))
	}

	// Validate role if provided
	if req.Role != nil {
		validRoles := map[models.UserRole]bool{
			models.RoleBuildingManager: true,
			models.RoleResident:        true,
			models.RoleSecurity:        true,
		}
		if !validRoles[*req.Role] {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid role"))
		}
	}

	user, err := h.userRepo.Update(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(user.ToResponse(), "Profile updated"))
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Current and new password are required"))
	}

	if len(req.NewPassword) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("New password must be at least 8 characters"))
	}

	// Verify current password
	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse("User not found"))
	}

	if err := services.ComparePassword(user.PasswordHash, req.CurrentPassword); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse("Current password is incorrect"))
	}

	// Hash new password
	newHash, err := services.HashPassword(req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse("Failed to hash password"))
	}

	if err := h.userRepo.UpdatePassword(c.Context(), userID, newHash); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Password changed successfully"))
}

// UpdateUserRole handles the request to change a user's role
func (h *AuthHandler) UpdateUserRole(c *fiber.Ctx) error {
	targetUserID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid user ID"))
	}

	var req struct {
		Role models.UserRole `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Role == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Role is required"))
	}

	if err := h.userRepo.UpdateRole(c.Context(), targetUserID, req.Role); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "User role updated"))
}
