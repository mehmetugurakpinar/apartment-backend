package handlers

import (
	"strconv"
	"time"

	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type TimelineHandler struct {
	timelineRepo *repository.TimelineRepository
	userRepo     *repository.UserRepository
	socialRepo   *repository.SocialRepository
}

func NewTimelineHandler(timelineRepo *repository.TimelineRepository, userRepo *repository.UserRepository, socialRepo *repository.SocialRepository) *TimelineHandler {
	return &TimelineHandler{timelineRepo: timelineRepo, userRepo: userRepo, socialRepo: socialRepo}
}

func (h *TimelineHandler) GetFeed(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var pq models.PaginationQuery
	c.QueryParser(&pq)
	pq.SetDefaults()

	buildingIDs, err := h.userRepo.GetBuildingsByUser(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	followedUserIDs, err := h.socialRepo.GetFollowedUserIDs(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	posts, total, err := h.timelineRepo.GetFeed(c.Context(), buildingIDs, followedUserIDs, userID, pq.Page, pq.Limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(models.NewPaginatedResponse(posts, pq.Page, pq.Limit, total), ""))
}

func (h *TimelineHandler) CreatePost(c *fiber.Ctx) error {
	userID := middleware.GetUserID(c)

	var req models.CreateTimelinePostRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	// Default type to "text" if not provided
	if req.Type == "" {
		req.Type = models.PostTypeText
	}
	// Default visibility to "building" if not provided
	if req.Visibility == "" {
		req.Visibility = models.VisibilityBuilding
	}

	// Get user's first building
	buildingIDs, err := h.userRepo.GetBuildingsByUser(c.Context(), userID)
	if err != nil || len(buildingIDs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("User has no building association"))
	}

	post := &models.TimelinePost{
		AuthorID:    userID,
		BuildingID:  buildingIDs[0],
		Content:     req.Content,
		Type:        req.Type,
		Visibility:  req.Visibility,
		LocationLat: req.LocationLat,
		LocationLng: req.LocationLng,
	}

	if err := h.timelineRepo.Create(c.Context(), post); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	// Create poll if it's a poll type
	if req.Type == models.PostTypePoll && req.Poll != nil {
		var endsAt *time.Time
		if req.Poll.EndsAt != nil {
			t, err := time.Parse(time.RFC3339, *req.Poll.EndsAt)
			if err == nil {
				endsAt = &t
			}
		}

		poll := &models.Poll{
			PostID:   post.ID,
			Question: req.Poll.Question,
			EndsAt:   endsAt,
		}

		if err := h.timelineRepo.CreatePoll(c.Context(), poll, req.Poll.Options); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
		}
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(post, "Post created"))
}

func (h *TimelineHandler) LikePost(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid post ID"))
	}

	userID := middleware.GetUserID(c)
	liked, err := h.timelineRepo.ToggleLike(c.Context(), postID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	msg := "Post unliked"
	if liked {
		msg = "Post liked"
	}
	return c.JSON(models.SuccessResponse(fiber.Map{"liked": liked}, msg))
}

func (h *TimelineHandler) CreateComment(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid post ID"))
	}

	var req models.CreateTimelineCommentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	userID := middleware.GetUserID(c)
	comment := &models.TimelineComment{
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

	if err := h.timelineRepo.CreateComment(c.Context(), comment); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(comment, "Comment added"))
}

func (h *TimelineHandler) VotePoll(c *fiber.Ctx) error {
	pollID, err := uuid.Parse(c.Params("pollId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid poll ID"))
	}

	var req models.PollVoteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid request body"))
	}

	optionID, err := uuid.Parse(req.OptionID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid option_id"))
	}

	userID := middleware.GetUserID(c)
	if err := h.timelineRepo.VotePoll(c.Context(), pollID, optionID, userID); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Vote recorded"))
}

func (h *TimelineHandler) RepostPost(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid post ID"))
	}

	userID := middleware.GetUserID(c)
	repost, err := h.timelineRepo.Repost(c.Context(), postID, userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse(err.Error()))
	}

	return c.Status(fiber.StatusCreated).JSON(models.SuccessResponse(repost, "Reposted"))
}

func (h *TimelineHandler) UnrepostPost(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid post ID"))
	}

	userID := middleware.GetUserID(c)
	if err := h.timelineRepo.Unrepost(c.Context(), postID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(nil, "Unreposted"))
}

func (h *TimelineHandler) GetPost(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid post ID"))
	}

	userID := middleware.GetUserID(c)
	post, err := h.timelineRepo.GetByID(c.Context(), postID, userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse("Post not found"))
	}

	return c.JSON(models.SuccessResponse(post, ""))
}

func (h *TimelineHandler) GetComments(c *fiber.Ctx) error {
	postID, err := uuid.Parse(c.Params("postId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("Invalid post ID"))
	}

	comments, err := h.timelineRepo.GetComments(c.Context(), postID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(comments, ""))
}

func (h *TimelineHandler) GetNearby(c *fiber.Ctx) error {
	lat, err := strconv.ParseFloat(c.Query("lat"), 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("lat is required"))
	}
	lng, err := strconv.ParseFloat(c.Query("lng"), 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse("lng is required"))
	}
	radius, _ := strconv.ParseFloat(c.Query("radius", "5"), 64)

	userID := middleware.GetUserID(c)
	posts, err := h.timelineRepo.GetNearby(c.Context(), lat, lng, radius, userID, 50)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse(err.Error()))
	}

	return c.JSON(models.SuccessResponse(posts, ""))
}
