package repository

import (
	"context"
	"fmt"

	"apartment-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (email, password_hash, phone, full_name, role)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		user.Email, user.PasswordHash, user.Phone, user.FullName, user.Role,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, phone, full_name, avatar_url, role, fcm_token, is_active, created_at, updated_at
		FROM users WHERE id = $1`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Phone,
		&user.FullName, &user.AvatarURL, &user.Role, &user.FCMToken,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `
		SELECT id, email, password_hash, phone, full_name, avatar_url, role, fcm_token, is_active, created_at, updated_at
		FROM users WHERE email = $1`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Phone,
		&user.FullName, &user.AvatarURL, &user.Role, &user.FCMToken,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateProfileRequest) (*models.User, error) {
	fmt.Printf("updating user=%s req=%+v\n", id, req)
	query := `
		UPDATE users SET
			full_name = COALESCE($2, full_name),
			phone = COALESCE($3, phone),
			avatar_url = COALESCE($4, avatar_url),
			fcm_token = COALESCE($5, fcm_token),
			role = COALESCE($6, role)
		WHERE id = $1
		RETURNING id, email, password_hash, phone, full_name, avatar_url, role, fcm_token, is_active, created_at, updated_at`

	user := &models.User{}
	err := r.pool.QueryRow(ctx, query, id, req.FullName, req.Phone, req.AvatarURL, req.FCMToken, req.Role).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Phone,
		&user.FullName, &user.AvatarURL, &user.Role, &user.FCMToken,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) UpdateRole(ctx context.Context, id uuid.UUID, role models.UserRole) error {
	query := `UPDATE users SET role = $2, updated_at = NOW() WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id, role)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, passwordHash)
	return err
}

func (r *UserRepository) SaveRefreshToken(ctx context.Context, rt *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query, rt.UserID, rt.TokenHash, rt.ExpiresAt).Scan(&rt.ID, &rt.CreatedAt)
}

func (r *UserRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens WHERE token_hash = $1 AND revoked = false`

	rt := &models.RefreshToken{}
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.Revoked, &rt.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return rt, nil
}

func (r *UserRepository) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked = true WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

func (r *UserRepository) RevokeAllRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked = true WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

func (r *UserRepository) GetBuildingsByUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT building_id FROM building_members WHERE user_id = $1`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buildingIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		buildingIDs = append(buildingIDs, id)
	}
	return buildingIDs, nil
}
