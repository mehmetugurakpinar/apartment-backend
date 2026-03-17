package models

import (
	"time"

	"github.com/google/uuid"
)

type CommonArea struct {
	ID               uuid.UUID `json:"id"`
	BuildingID       uuid.UUID `json:"building_id"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	Capacity         int       `json:"capacity"`
	Rules            string    `json:"rules,omitempty"`
	OpenTime         string    `json:"open_time"`
	CloseTime        string    `json:"close_time"`
	RequiresApproval bool      `json:"requires_approval"`
	IsActive         bool      `json:"is_active"`
	ImageURL         string    `json:"image_url,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type Reservation struct {
	ID           uuid.UUID `json:"id"`
	CommonAreaID uuid.UUID `json:"common_area_id"`
	UserID       uuid.UUID `json:"user_id"`
	BuildingID   uuid.UUID `json:"building_id"`
	Title        string    `json:"title,omitempty"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	GuestCount   int       `json:"guest_count"`
	Status       string    `json:"status"`
	Notes        string    `json:"notes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Joined fields
	AreaName string `json:"area_name,omitempty"`
	UserName string `json:"user_name,omitempty"`
}

type CreateCommonAreaRequest struct {
	Name             string `json:"name" validate:"required"`
	Description      string `json:"description,omitempty"`
	Capacity         int    `json:"capacity,omitempty"`
	Rules            string `json:"rules,omitempty"`
	OpenTime         string `json:"open_time,omitempty"`
	CloseTime        string `json:"close_time,omitempty"`
	RequiresApproval bool   `json:"requires_approval,omitempty"`
}

type CreateReservationRequest struct {
	CommonAreaID string `json:"common_area_id" validate:"required"`
	Title        string `json:"title,omitempty"`
	StartTime    string `json:"start_time" validate:"required"`
	EndTime      string `json:"end_time" validate:"required"`
	GuestCount   int    `json:"guest_count,omitempty"`
	Notes        string `json:"notes,omitempty"`
}
