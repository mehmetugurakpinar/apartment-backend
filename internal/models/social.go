package models

import (
	"time"

	"github.com/google/uuid"
)

type UserFollow struct {
	ID          uuid.UUID `json:"id"`
	FollowerID  uuid.UUID `json:"follower_id"`
	FollowingID uuid.UUID `json:"following_id"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserSearchResult struct {
	ID             uuid.UUID `json:"id"`
	FullName       string    `json:"full_name"`
	AvatarURL      *string   `json:"avatar_url,omitempty"`
	IsFollowing    bool      `json:"is_following"`
	FollowerCount  int       `json:"follower_count"`
	FollowingCount int       `json:"following_count"`
}

type UserProfileResponse struct {
	ID             uuid.UUID `json:"id"`
	FullName       string    `json:"full_name"`
	AvatarURL      *string   `json:"avatar_url,omitempty"`
	FollowerCount  int       `json:"follower_count"`
	FollowingCount int       `json:"following_count"`
	IsFollowing    bool      `json:"is_following"`
}
