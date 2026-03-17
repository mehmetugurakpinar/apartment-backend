package handlers

import (
	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type SocialHandler struct {
	socialRepo *repository.SocialRepository
}

func NewSocialHandler(socialRepo *repository.SocialRepository) *SocialHandler {
	return &SocialHandler{socialRepo: socialRepo}
}

func (h *SocialHandler) FollowUser(c *fiber.Ctx) error {
	targetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid user ID"))
	}

	userID := middleware.GetUserID(c)
	if userID == targetID {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Cannot follow yourself"))
	}

	if err := h.socialRepo.FollowUser(c.Context(), userID, targetID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Followed"))
}

func (h *SocialHandler) UnfollowUser(c *fiber.Ctx) error {
	targetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid user ID"))
	}

	userID := middleware.GetUserID(c)
	if err := h.socialRepo.UnfollowUser(c.Context(), userID, targetID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Unfollowed"))
}

func (h *SocialHandler) GetFollowers(c *fiber.Ctx) error {
	targetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid user ID"))
	}

	var pq models.PaginationQuery
	c.QueryParser(&pq)
	pq.SetDefaults()

	userID := middleware.GetUserID(c)
	users, total, err := h.socialRepo.GetFollowers(c.Context(), targetID, userID, pq.Page, pq.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(models.NewPaginatedResponse(users, pq.Page, pq.Limit, total), ""))
}

func (h *SocialHandler) GetFollowing(c *fiber.Ctx) error {
	targetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid user ID"))
	}

	var pq models.PaginationQuery
	c.QueryParser(&pq)
	pq.SetDefaults()

	userID := middleware.GetUserID(c)
	users, total, err := h.socialRepo.GetFollowing(c.Context(), targetID, userID, pq.Page, pq.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(models.NewPaginatedResponse(users, pq.Page, pq.Limit, total), ""))
}

func (h *SocialHandler) GetUserProfile(c *fiber.Ctx) error {
	targetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid user ID"))
	}

	userID := middleware.GetUserID(c)
	profile, err := h.socialRepo.GetUserProfile(c.Context(), targetID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse("User not found"))
	}

	return c.JSON(models.SuccessResponse(profile, ""))
}

func (h *SocialHandler) SearchUsers(c *fiber.Ctx) error {
	q := c.Query("q")
	if q == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Query parameter 'q' is required"))
	}

	userID := middleware.GetUserID(c)
	users, err := h.socialRepo.SearchUsers(c.Context(), q, userID, 20)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(users, ""))
}
