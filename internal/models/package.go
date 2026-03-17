package models

import (
	"time"

	"github.com/google/uuid"
)

type Package struct {
	ID             uuid.UUID  `json:"id"`
	BuildingID     uuid.UUID  `json:"building_id"`
	UnitID         *uuid.UUID `json:"unit_id,omitempty"`
	RecipientID    *uuid.UUID `json:"recipient_id,omitempty"`
	Carrier        string     `json:"carrier,omitempty"`
	TrackingNumber string     `json:"tracking_number,omitempty"`
	Description    string     `json:"description,omitempty"`
	ReceivedBy     *uuid.UUID `json:"received_by,omitempty"`
	ReceivedAt     time.Time  `json:"received_at"`
	PickedUpBy     *uuid.UUID `json:"picked_up_by,omitempty"`
	PickedUpAt     *time.Time `json:"picked_up_at,omitempty"`
	PhotoURL       string     `json:"photo_url,omitempty"`
	Status         string     `json:"status"`
	Notes          string     `json:"notes,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	// Joined fields
	RecipientName  string `json:"recipient_name,omitempty"`
	UnitNumber     string `json:"unit_number,omitempty"`
	ReceivedByName string `json:"received_by_name,omitempty"`
}

type CreatePackageRequest struct {
	RecipientID    string `json:"recipient_id,omitempty"`
	UnitNumber     string `json:"unit_number,omitempty"`
	Carrier        string `json:"carrier,omitempty"`
	TrackingNumber string `json:"tracking_number,omitempty"`
	Description    string `json:"description,omitempty"`
	Notes          string `json:"notes,omitempty"`
}
