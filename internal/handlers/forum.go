package handlers

import (
	"fmt"

	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"
	"apartment-backend/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ForumHandler struct {
	forumRepo      *repository.ForumRepository
	buildingRepo   *repository.BuildingRepository
	storageService *services.StorageService
}

func NewForumHandler(forumRepo *repository.ForumRepository, buildingRepo *repository.BuildingRepository, storageService *services.StorageService) *ForumHandler {
	return &ForumHandler{forumRepo: forumRepo, buildingRepo: buildingRepo, storageService: storageService}
}

var errForbidden = fmt.Errorf("forbidden")

func (h *ForumHandler) checkBuildingAccess(c *fiber.Ctx, buildingID uuid.UUID) error {
	userID := middleware.GetUserID(c)
	isMember, err := h.buildingRepo.IsMember(c.Context(), buildingID, userID)
	if err != nil {
		c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
		return err
	}
	if !isMember {
		c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse("You are not a member of this building"))
		return errForbidden
	}
	return nil
}

func (h *ForumHandler) GetCategories(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}
	if err := h.checkBuildingAccess(c, buildingID); err != nil {
		return nil
	}

	categories, err := h.forumRepo.GetCategories(c.Context(), buildingID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(categories, ""))
}

func (h *ForumHandler) GetPosts(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}
	if err := h.checkBuildingAccess(c, buildingID); err != nil {
		return nil
	}

	var pq models.PaginationQuery
	c.QueryParser(&pq)
	pq.SetDefaults()

	var categoryID *uuid.UUID
	if catStr := c.Query("category_id"); catStr != "" {
		id, err := uuid.Parse(catStr)
		if err == nil {
			categoryID = &id
		}
	}

	posts, total, err := h.forumRepo.GetPosts(c.Context(), buildingID, categoryID, pq.Page, pq.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(models.NewPaginatedResponse(posts, pq.Page, pq.Limit, total), ""))
}

func (h *ForumHandler) CreatePost(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}
	if err := h.checkBuildingAccess(c, buildingID); err != nil {
		return nil
	}

	var req models.CreateForumPostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Title == "" || req.Body == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Title and body are required"))
	}

	// category_id is optional – if missing, use or create "General"
	var categoryID uuid.UUID
	if req.CategoryID != "" {
		parsed, err := uuid.Parse(req.CategoryID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid category_id"))
		}
		categoryID = parsed
	} else {
		// Get or create default "General" category for this building
		cat, err := h.forumRepo.GetOrCreateDefaultCategory(c.Context(), buildingID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
		}
		categoryID = cat
	}

	userID := middleware.GetUserID(c)
	post := &models.ForumPost{
		BuildingID: buildingID,
		CategoryID: categoryID,
		AuthorID:   userID,
		Title:      req.Title,
		Body:       req.Body,
	}

	if err := h.forumRepo.CreatePost(c.Context(), post); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(post, "Post created"))
}

func (h *ForumHandler) GetPost(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid post ID"))
	}

	userID := middleware.GetUserID(c)
	post, err := h.forumRepo.GetPostByID(c.Context(), postID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(post, ""))
}

func (h *ForumHandler) CreateComment(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid post ID"))
	}

	var req models.CreateForumCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	userID := middleware.GetUserID(c)
	comment := &models.ForumComment{
		PostID:   postID,
		AuthorID: userID,
		Body:     req.Body,
	}

	if req.ParentID != nil {
		parentID, err := uuid.Parse(*req.ParentID)
		if err == nil {
			comment.ParentID = &parentID
		}
	}

	if err := h.forumRepo.CreateComment(c.Context(), comment); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(comment, "Comment added"))
}

func (h *ForumHandler) Vote(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid post ID"))
	}

	var req models.VoteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	if req.Value != 1 && req.Value != -1 {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Vote value must be 1 or -1"))
	}

	userID := middleware.GetUserID(c)
	vote := &models.ForumVote{
		PostID: postID,
		UserID: userID,
		Value:  req.Value,
	}

	if err := h.forumRepo.Vote(c.Context(), vote); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Vote recorded"))
}

func (h *ForumHandler) UploadMedia(c *fiber.Ctx) error {
	buildingID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid building ID"))
	}
	if err := h.checkBuildingAccess(c, buildingID); err != nil {
		return nil
	}

	postID, err := uuid.Parse(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid post ID"))
	}

	if h.storageService == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.ErrorResponse("File storage not configured"))
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("No file provided"))
	}

	// Validate content type
	ct := file.Header.Get("Content-Type")
	switch ct {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
	default:
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Only image files are allowed (jpeg, png, gif, webp)"))
	}

	// 5MB limit
	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("File size must be under 5MB"))
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse("Failed to read file"))
	}
	defer f.Close()

	url, err := h.storageService.UploadFile(c.Context(), f, file.Size, ct, "forum")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse("Failed to upload file"))
	}

	media, err := h.forumRepo.AddMedia(c.Context(), postID, url, "image")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(media, "Media uploaded"))
}
