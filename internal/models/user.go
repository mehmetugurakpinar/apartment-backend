package models

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleSuperAdmin      UserRole = "super_admin"
	RoleBuildingManager UserRole = "building_manager"
	RoleResident        UserRole = "resident"
	RoleSecurity        UserRole = "security"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Phone        *string   `json:"phone,omitempty"`
	FullName     string    `json:"full_name"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	Role         UserRole  `json:"role"`
	FCMToken     *string   `json:"-"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RefreshToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"created_at"`
}

// Request/Response DTOs

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	FullName string `json:"full_name" validate:"required,min=2"`
	Phone    string `json:"phone,omitempty"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=8"`
}

type UpdateProfileRequest struct {
	FullName  *string   `json:"full_name,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	FCMToken  *string   `json:"fcm_token,omitempty"`
	Role      *UserRole `json:"role,omitempty"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    time.Time    `json:"expires_at"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Phone     *string   `json:"phone,omitempty"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	Role      UserRole  `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		FullName:  u.FullName,
		Phone:     u.Phone,
		AvatarURL: u.AvatarURL,
		Role:      u.Role,
		CreatedAt: u.CreatedAt,
	}
}
