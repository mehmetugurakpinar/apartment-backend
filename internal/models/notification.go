package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID         uuid.UUID        `json:"id"`
	UserID     uuid.UUID        `json:"user_id"`
	BuildingID *uuid.UUID       `json:"building_id,omitempty"`
	Type       string           `json:"type"`
	Title      string           `json:"title"`
	Body       *string          `json:"body,omitempty"`
	Data       *json.RawMessage `json:"data,omitempty"`
	ReadAt     *time.Time       `json:"read_at,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
}

type NotificationPreference struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Category     string    `json:"category"`
	PushEnabled  bool      `json:"push_enabled"`
	EmailEnabled bool      `json:"email_enabled"`
}

// Request DTOs

type CreateAnnouncementRequest struct {
	Title string `json:"title" validate:"required"`
	Body  string `json:"body" validate:"required"`
	Type  string `json:"type" validate:"required"`
}

type UpdatePreferencesRequest struct {
	Category     string `json:"category" validate:"required"`
	PushEnabled  *bool  `json:"push_enabled,omitempty"`
	EmailEnabled *bool  `json:"email_enabled,omitempty"`
}
