package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"apartment-backend/internal/config"
	"apartment-backend/internal/middleware"
	"apartment-backend/internal/models"
	"apartment-backend/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
}

func NewAuthService(userRepo *repository.UserRepository, cfg *config.Config) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ComparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Check if email exists
	existing, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("email already registered")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &models.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		Role:         models.RoleResident,
		IsActive:     true,
	}
	if req.Phone != "" {
		user.Phone = &req.Phone
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return s.generateAuthResponse(ctx, user)
}

func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return s.generateAuthResponse(ctx, user)
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*models.AuthResponse, error) {
	userID, err := middleware.ParseRefreshToken(s.cfg.JWT, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Verify refresh token in DB
	tokenHash := hashToken(refreshToken)
	storedToken, err := s.userRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if storedToken == nil {
		return nil, fmt.Errorf("refresh token not found or revoked")
	}

	// Revoke old refresh token (rotation)
	if err := s.userRepo.RevokeRefreshToken(ctx, storedToken.ID); err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return s.generateAuthResponse(ctx, user)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)
	storedToken, err := s.userRepo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		return err
	}
	if storedToken != nil {
		return s.userRepo.RevokeRefreshToken(ctx, storedToken.ID)
	}
	return nil
}

func (s *AuthService) GetProfile(ctx context.Context, userID interface{}) (*models.User, error) {
	id, ok := userID.(interface{ String() string })
	if !ok {
		return nil, fmt.Errorf("invalid user ID")
	}
	_ = id
	return nil, fmt.Errorf("not implemented via this path")
}

func (s *AuthService) generateAuthResponse(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
	accessToken, expiresAt, err := middleware.GenerateAccessToken(s.cfg.JWT, user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, refreshExpires, err := middleware.GenerateRefreshToken(s.cfg.JWT, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token
	tokenHash := hashToken(refreshToken)
	rt := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: refreshExpires,
	}
	if err := s.userRepo.SaveRefreshToken(ctx, rt); err != nil {
		return nil, fmt.Errorf("failed to save refresh token: %w", err)
	}

	return &models.AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
