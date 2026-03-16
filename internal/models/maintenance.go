package models

import (
	"time"

	"github.com/google/uuid"
)

type MaintenancePriority string

const (
	PriorityEmergency MaintenancePriority = "emergency"
	PriorityHigh      MaintenancePriority = "high"
	PriorityNormal    MaintenancePriority = "normal"
	PriorityLow       MaintenancePriority = "low"
)

type MaintenanceRequestStatus string

const (
	MaintenancePendingApproval MaintenanceRequestStatus = "pending_approval"
	MaintenanceOpen            MaintenanceRequestStatus = "open"
	MaintenanceInProgress      MaintenanceRequestStatus = "in_progress"
	MaintenanceResolved        MaintenanceRequestStatus = "resolved"
	MaintenanceClosed          MaintenanceRequestStatus = "closed"
)

type MaintenanceRequest struct {
	ID          uuid.UUID                `json:"id"`
	BuildingID  uuid.UUID                `json:"building_id"`
	UnitID      *uuid.UUID               `json:"unit_id,omitempty"`
	Title       string                   `json:"title"`
	Description *string                  `json:"description,omitempty"`
	Priority    MaintenancePriority      `json:"priority"`
	Status      MaintenanceRequestStatus `json:"status"`
	AssignedTo  *uuid.UUID               `json:"assigned_to,omitempty"`
	CreatedBy   uuid.UUID                `json:"created_by"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
	ResolvedAt  *time.Time               `json:"resolved_at,omitempty"`
}

type MaintenancePhoto struct {
	ID        uuid.UUID `json:"id"`
	RequestID uuid.UUID `json:"request_id"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type Vendor struct {
	ID         uuid.UUID `json:"id"`
	BuildingID uuid.UUID `json:"building_id"`
	Name       string    `json:"name"`
	Category   *string   `json:"category,omitempty"`
	Phone      *string   `json:"phone,omitempty"`
	Email      *string   `json:"email,omitempty"`
	Rating     float64   `json:"rating"`
	CreatedAt  time.Time `json:"created_at"`
}

// Request DTOs

type CreateMaintenanceRequest struct {
	UnitID      *string             `json:"unit_id,omitempty"`
	Title       string              `json:"title" validate:"required"`
	Description *string             `json:"description,omitempty"`
	Priority    MaintenancePriority `json:"priority" validate:"required"`
}

type UpdateMaintenanceRequest struct {
	Title       *string                   `json:"title,omitempty"`
	Description *string                   `json:"description,omitempty"`
	Status      *MaintenanceRequestStatus `json:"status,omitempty"`
	Priority    *MaintenancePriority      `json:"priority,omitempty"`
	AssignedTo  *string                   `json:"assigned_to,omitempty"`
}

type CreateVendorRequest struct {
	Name     string  `json:"name" validate:"required"`
	Category *string `json:"category,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Email    *string `json:"email,omitempty"`
}

type UpdateVendorRequest struct {
	Name     *string `json:"name,omitempty"`
	Category *string `json:"category,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Email    *string `json:"email,omitempty"`
}

// Response DTOs

type MaintenanceRequestDetail struct {
	MaintenanceRequest
	Photos         []MaintenancePhoto `json:"photos"`
	CreatedByName  string             `json:"created_by_name"`
	AssignedToName *string            `json:"assigned_to_name,omitempty"`
	UnitNumber     *string            `json:"unit_number,omitempty"`
}
