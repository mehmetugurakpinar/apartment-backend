package models

import (
	"time"

	"github.com/google/uuid"
)

type TimelinePostType string

const (
	PostTypeText  TimelinePostType = "text"
	PostTypePhoto TimelinePostType = "photo"
	PostTypePoll  TimelinePostType = "poll"
	PostTypeEvent TimelinePostType = "event"
	PostTypeAlert TimelinePostType = "alert"
)

type VisibilityType string

const (
	VisibilityBuilding     VisibilityType = "building"
	VisibilityNeighborhood VisibilityType = "neighborhood"
	VisibilityPublic       VisibilityType = "public"
)

type TimelinePost struct {
	ID             uuid.UUID        `json:"id"`
	AuthorID       uuid.UUID        `json:"author_id"`
	BuildingID     uuid.UUID        `json:"building_id"`
	Content        *string          `json:"content,omitempty"`
	Type           TimelinePostType `json:"type"`
	Visibility     VisibilityType   `json:"visibility"`
	LocationLat    *float64         `json:"location_lat,omitempty"`
	LocationLng    *float64         `json:"location_lng,omitempty"`
	LikeCount      int              `json:"like_count"`
	CommentCount   int              `json:"comment_count"`
	RepostCount    int              `json:"repost_count"`
	OriginalPostID *uuid.UUID       `json:"original_post_id,omitempty"`
	IsRepost       bool             `json:"is_repost"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type TimelineMedia struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	URL       string    `json:"url"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type TimelineLike struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type TimelineComment struct {
	ID        uuid.UUID  `json:"id"`
	PostID    uuid.UUID  `json:"post_id"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	AuthorID  uuid.UUID  `json:"author_id"`
	Body      string     `json:"body"`
	CreatedAt time.Time  `json:"created_at"`
}

type Poll struct {
	ID         uuid.UUID  `json:"id"`
	PostID     uuid.UUID  `json:"post_id"`
	Question   string     `json:"question"`
	EndsAt     *time.Time `json:"ends_at,omitempty"`
	TotalVotes int        `json:"total_votes"`
	CreatedAt  time.Time  `json:"created_at"`
}

type PollOption struct {
	ID        uuid.UUID `json:"id"`
	PollID    uuid.UUID `json:"poll_id"`
	Text      string    `json:"text"`
	VoteCount int       `json:"vote_count"`
	SortOrder int       `json:"sort_order"`
}

type PollVote struct {
	ID        uuid.UUID `json:"id"`
	PollID    uuid.UUID `json:"poll_id"`
	OptionID  uuid.UUID `json:"option_id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// Request DTOs

type CreateTimelinePostRequest struct {
	Content     *string            `json:"content,omitempty"`
	Type        TimelinePostType   `json:"type" validate:"required"`
	Visibility  VisibilityType     `json:"visibility" validate:"required"`
	LocationLat *float64           `json:"location_lat,omitempty"`
	LocationLng *float64           `json:"location_lng,omitempty"`
	Poll        *CreatePollRequest `json:"poll,omitempty"`
}

type CreatePollRequest struct {
	Question string   `json:"question" validate:"required"`
	Options  []string `json:"options" validate:"required,min=2"`
	EndsAt   *string  `json:"ends_at,omitempty"`
}

type CreateTimelineCommentRequest struct {
	ParentID *string `json:"parent_id,omitempty"`
	Body     string  `json:"body" validate:"required"`
}

type PollVoteRequest struct {
	OptionID string `json:"option_id" validate:"required"`
}

// Response DTOs

type TimelinePostDetail struct {
	TimelinePost
	AuthorName         string          `json:"author_name"`
	AuthorAvatar       *string         `json:"author_avatar,omitempty"`
	Media              []TimelineMedia `json:"media,omitempty"`
	Poll               *PollDetail     `json:"poll,omitempty"`
	IsLiked            bool            `json:"is_liked"`
	IsReposted         bool            `json:"is_reposted"`
	OriginalAuthorName *string         `json:"original_author_name,omitempty"`
}

type PollDetail struct {
	Poll
	Options  []PollOption `json:"options"`
	UserVote *uuid.UUID   `json:"user_vote,omitempty"`
}

type TimelineCommentDetail struct {
	TimelineComment
	AuthorName   string  `json:"author_name"`
	AuthorAvatar *string `json:"author_avatar,omitempty"`
}
