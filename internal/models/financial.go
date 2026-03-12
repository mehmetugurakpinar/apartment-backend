package models

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentPending PaymentStatus = "pending"
	PaymentPaid    PaymentStatus = "paid"
	PaymentLate    PaymentStatus = "late"
)

type DuesPlan struct {
	ID          uuid.UUID  `json:"id"`
	BuildingID  uuid.UUID  `json:"building_id"`
	Title       string     `json:"title"`
	Amount      float64    `json:"amount"`
	PeriodMonth int        `json:"period_month"`
	PeriodYear  int        `json:"period_year"`
	DueDate     time.Time  `json:"due_date"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type DuePayment struct {
	ID         uuid.UUID     `json:"id"`
	DuesPlanID uuid.UUID     `json:"dues_plan_id"`
	UnitID     uuid.UUID     `json:"unit_id"`
	PaidAmount *float64      `json:"paid_amount,omitempty"`
	PaidAt     *time.Time    `json:"paid_at,omitempty"`
	ReceiptURL *string       `json:"receipt_url,omitempty"`
	Status     PaymentStatus `json:"status"`
	Notes      *string       `json:"notes,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
}

type Expense struct {
	ID          uuid.UUID  `json:"id"`
	BuildingID  uuid.UUID  `json:"building_id"`
	Category    string     `json:"category"`
	Amount      float64    `json:"amount"`
	Description *string    `json:"description,omitempty"`
	Date        time.Time  `json:"date"`
	ReceiptURL  *string    `json:"receipt_url,omitempty"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// Request DTOs

type CreateDuesPlanRequest struct {
	Title       string  `json:"title" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	PeriodMonth int     `json:"period_month" validate:"required,min=1,max=12"`
	PeriodYear  int     `json:"period_year" validate:"required,min=2020"`
	DueDate     string  `json:"due_date" validate:"required"`
}

type PayDuesRequest struct {
	UnitID     string  `json:"unit_id" validate:"required"`
	PaidAmount float64 `json:"paid_amount" validate:"required,gt=0"`
	Notes      *string `json:"notes,omitempty"`
}

type CreateExpenseRequest struct {
	Category    string  `json:"category" validate:"required"`
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Description *string `json:"description,omitempty"`
	Date        string  `json:"date" validate:"required"`
}

type UpdateDuesPlanRequest struct {
	Title       *string  `json:"title,omitempty"`
	Amount      *float64 `json:"amount,omitempty"`
	PeriodMonth *int     `json:"period_month,omitempty"`
	PeriodYear  *int     `json:"period_year,omitempty"`
	DueDate     *string  `json:"due_date,omitempty"`
}

type UpdateExpenseRequest struct {
	Category    *string  `json:"category,omitempty"`
	Amount      *float64 `json:"amount,omitempty"`
	Description *string  `json:"description,omitempty"`
	Date        *string  `json:"date,omitempty"`
}

// Response DTOs

type DuesReport struct {
	BuildingID   uuid.UUID          `json:"building_id"`
	PeriodMonth  int                `json:"period_month"`
	PeriodYear   int                `json:"period_year"`
	TotalDue     float64            `json:"total_due"`
	TotalPaid    float64            `json:"total_paid"`
	TotalPending float64            `json:"total_pending"`
	PaidCount    int                `json:"paid_count"`
	PendingCount int                `json:"pending_count"`
	LateCount    int                `json:"late_count"`
	Payments     []DuePaymentDetail `json:"payments"`
}

type DuePaymentDetail struct {
	DuePayment
	UnitNumber string `json:"unit_number"`
	OwnerName  string `json:"owner_name"`
}
