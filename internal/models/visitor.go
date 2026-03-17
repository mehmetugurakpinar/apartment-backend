package models

import (
	"time"

	"github.com/google/uuid"
)

type VisitorPass struct {
	ID           uuid.UUID  `json:"id"`
	BuildingID   uuid.UUID  `json:"building_id"`
	UnitID       *uuid.UUID `json:"unit_id,omitempty"`
	CreatedBy    uuid.UUID  `json:"created_by"`
	VisitorName  string     `json:"visitor_name"`
	VisitorPhone string     `json:"visitor_phone,omitempty"`
	VisitorPlate string     `json:"visitor_plate,omitempty"`
	Purpose      string     `json:"purpose,omitempty"`
	ExpectedAt   *time.Time `json:"expected_at,omitempty"`
	CheckedInAt  *time.Time `json:"checked_in_at,omitempty"`
	CheckedOutAt *time.Time `json:"checked_out_at,omitempty"`
	QRCode       string     `json:"qr_code,omitempty"`
	Status       string     `json:"status"`
	ApprovedBy   *uuid.UUID `json:"approved_by,omitempty"`
	Notes        string     `json:"notes,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Joined fields
	CreatedByName string `json:"created_by_name,omitempty"`
	UnitNumber    string `json:"unit_number,omitempty"`
}

type CreateVisitorRequest struct {
	VisitorName  string `json:"visitor_name" validate:"required"`
	VisitorPhone string `json:"visitor_phone,omitempty"`
	VisitorPlate string `json:"visitor_plate,omitempty"`
	Purpose      string `json:"purpose,omitempty"`
	ExpectedAt   string `json:"expected_at,omitempty"`
	Notes        string `json:"notes,omitempty"`
}
