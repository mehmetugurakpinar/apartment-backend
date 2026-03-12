package models

import (
	"time"

	"github.com/google/uuid"
)

type UnitStatus string

const (
	UnitOccupied    UnitStatus = "occupied"
	UnitVacant      UnitStatus = "vacant"
	UnitMaintenance UnitStatus = "maintenance"
)

type ResidentType string

const (
	ResidentOwner  ResidentType = "owner"
	ResidentTenant ResidentType = "tenant"
)

type Building struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Address    string     `json:"address"`
	City       string     `json:"city"`
	TotalUnits int        `json:"total_units"`
	Latitude   *float64   `json:"latitude,omitempty"`
	Longitude  *float64   `json:"longitude,omitempty"`
	ImageURL   *string    `json:"image_url,omitempty"`
	CreatedBy  *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type Unit struct {
	ID         uuid.UUID  `json:"id"`
	BuildingID uuid.UUID  `json:"building_id"`
	Block      *string    `json:"block,omitempty"`
	Floor      int        `json:"floor"`
	UnitNumber string     `json:"unit_number"`
	AreaSqm    *float64   `json:"area_sqm,omitempty"`
	OwnerID    *uuid.UUID `json:"owner_id,omitempty"`
	Status     UnitStatus `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type UnitResident struct {
	ID          uuid.UUID    `json:"id"`
	UnitID      uuid.UUID    `json:"unit_id"`
	UserID      uuid.UUID    `json:"user_id"`
	Type        ResidentType `json:"type"`
	MoveInDate  time.Time    `json:"move_in_date"`
	MoveOutDate *time.Time   `json:"move_out_date,omitempty"`
	IsActive    bool         `json:"is_active"`
	CreatedAt   time.Time    `json:"created_at"`
}

type Vehicle struct {
	ID          uuid.UUID `json:"id"`
	UnitID      uuid.UUID `json:"unit_id"`
	Plate       string    `json:"plate"`
	Type        *string   `json:"type,omitempty"`
	ParkingSpot *string   `json:"parking_spot,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type BuildingMember struct {
	ID         uuid.UUID `json:"id"`
	BuildingID uuid.UUID `json:"building_id"`
	UserID     uuid.UUID `json:"user_id"`
	Role       UserRole  `json:"role"`
	JoinedAt   time.Time `json:"joined_at"`
}

// Request DTOs

type CreateBuildingRequest struct {
	Name       string   `json:"name" validate:"required"`
	Address    string   `json:"address" validate:"required"`
	City       string   `json:"city" validate:"required"`
	TotalUnits int      `json:"total_units" validate:"required,min=1"`
	Latitude   *float64 `json:"latitude,omitempty"`
	Longitude  *float64 `json:"longitude,omitempty"`
}

type CreateUnitRequest struct {
	Block      *string  `json:"block,omitempty"`
	Floor      int      `json:"floor" validate:"required"`
	UnitNumber string   `json:"unit_number" validate:"required"`
	AreaSqm    *float64 `json:"area_sqm,omitempty"`
	OwnerID    *string  `json:"owner_id,omitempty"`
}

type UpdateUnitRequest struct {
	Block      *string     `json:"block,omitempty"`
	Floor      *int        `json:"floor,omitempty"`
	UnitNumber *string     `json:"unit_number,omitempty"`
	AreaSqm    *float64    `json:"area_sqm,omitempty"`
	OwnerID    *string     `json:"owner_id,omitempty"`
	Status     *UnitStatus `json:"status,omitempty"`
}

type AddResidentRequest struct {
	UserID     string       `json:"user_id" validate:"required"`
	Type       ResidentType `json:"type" validate:"required"`
	MoveInDate string       `json:"move_in_date,omitempty"`
}

// Response DTOs

type BuildingDashboard struct {
	Building        Building `json:"building"`
	TotalUnits      int      `json:"total_units"`
	OccupiedUnits   int      `json:"occupied_units"`
	VacantUnits     int      `json:"vacant_units"`
	TotalResidents  int      `json:"total_residents"`
	PendingDues     int      `json:"pending_dues"`
	OpenRequests    int      `json:"open_requests"`
	MonthlyIncome   float64  `json:"monthly_income"`
	MonthlyExpenses float64  `json:"monthly_expenses"`
}

type UnitWithResidents struct {
	Unit      Unit           `json:"unit"`
	Owner     *UserResponse  `json:"owner,omitempty"`
	Residents []UserResponse `json:"residents"`
}

type BuildingMemberDetail struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Phone     *string   `json:"phone,omitempty"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	Role      UserRole  `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}

// Invitation models

type BuildingInvitation struct {
	ID         uuid.UUID  `json:"id"`
	BuildingID uuid.UUID  `json:"building_id"`
	Email      string     `json:"email"`
	Role       UserRole   `json:"role"`
	Token      string     `json:"token"`
	InvitedBy  uuid.UUID  `json:"invited_by"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CreateInvitationRequest struct {
	Email string   `json:"email" validate:"required,email"`
	Role  UserRole `json:"role" validate:"required"`
}

type AcceptInvitationRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required"`
}
