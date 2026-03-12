package models

import (
	"time"

	"github.com/google/uuid"
)

type ForumCategory struct {
	ID         uuid.UUID `json:"id"`
	BuildingID uuid.UUID `json:"building_id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	SortOrder  int       `json:"sort_order"`
	CreatedAt  time.Time `json:"created_at"`
}

type ForumPost struct {
	ID           uuid.UUID `json:"id"`
	BuildingID   uuid.UUID `json:"building_id"`
	CategoryID   uuid.UUID `json:"category_id"`
	AuthorID     uuid.UUID `json:"author_id"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	IsPinned     bool      `json:"is_pinned"`
	IsResolved   bool      `json:"is_resolved"`
	Upvotes      int       `json:"upvotes"`
	Downvotes    int       `json:"downvotes"`
	CommentCount int       `json:"comment_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type ForumComment struct {
	ID        uuid.UUID  `json:"id"`
	PostID    uuid.UUID  `json:"post_id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	AuthorID  uuid.UUID  `json:"author_id"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ForumVote struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	UserID    uuid.UUID `json:"user_id"`
	Value     int       `json:"value"` // 1 or -1
	CreatedAt time.Time `json:"created_at"`
}

// Request DTOs

type CreateForumPostRequest struct {
	CategoryID string `json:"category_id,omitempty"`
	Title      string `json:"title" validate:"required"`
	Body       string `json:"body" validate:"required"`
}

type CreateForumCommentRequest struct {
	ParentID *string `json:"parent_id,omitempty"`
	Body     string  `json:"body" validate:"required"`
}

type VoteRequest struct {
	Value int `json:"value" validate:"required,oneof=-1 1"`
}

// Response DTOs

type ForumPostDetail struct {
	ForumPost
	AuthorName   string          `json:"author_name"`
	AuthorAvatar *string         `json:"author_avatar,omitempty"`
	CategoryName string          `json:"category_name"`
	Comments     []CommentDetail `json:"comments,omitempty"`
	UserVote     *int            `json:"user_vote,omitempty"`
}

type CommentDetail struct {
	ForumComment
	AuthorName   string  `json:"author_name"`
	AuthorAvatar *string `json:"author_avatar,omitempty"`
}
