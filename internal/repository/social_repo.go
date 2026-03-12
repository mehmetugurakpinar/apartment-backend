package repository

import (
	"context"

	"apartment-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SocialRepository struct {
	pool *pgxpool.Pool
}

func NewSocialRepository(pool *pgxpool.Pool) *SocialRepository {
	return &SocialRepository{pool: pool}
}

func (r *SocialRepository) FollowUser(ctx context.Context, followerID, followingID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO user_follows (follower_id, following_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		followerID, followingID,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE users SET following_count = following_count + 1 WHERE id = $1`, followerID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE users SET follower_count = follower_count + 1 WHERE id = $1`, followingID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *SocialRepository) UnfollowUser(ctx context.Context, followerID, followingID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	result, err := tx.Exec(ctx,
		`DELETE FROM user_follows WHERE follower_id = $1 AND following_id = $2`,
		followerID, followingID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return nil // Not following, no-op
	}

	_, err = tx.Exec(ctx, `UPDATE users SET following_count = GREATEST(following_count - 1, 0) WHERE id = $1`, followerID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `UPDATE users SET follower_count = GREATEST(follower_count - 1, 0) WHERE id = $1`, followingID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *SocialRepository) IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_follows WHERE follower_id = $1 AND following_id = $2)`,
		followerID, followingID,
	).Scan(&exists)
	return exists, err
}

func (r *SocialRepository) GetFollowers(ctx context.Context, userID uuid.UUID, requestingUserID uuid.UUID, page, limit int) ([]models.UserSearchResult, int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_follows WHERE following_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query := `
		SELECT u.id, u.full_name, u.avatar_url, u.follower_count, u.following_count,
			EXISTS(SELECT 1 FROM user_follows WHERE follower_id = $2 AND following_id = u.id) as is_following
		FROM user_follows uf
		JOIN users u ON uf.follower_id = u.id
		WHERE uf.following_id = $1
		ORDER BY uf.created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, query, userID, requestingUserID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.UserSearchResult
	for rows.Next() {
		var u models.UserSearchResult
		if err := rows.Scan(&u.ID, &u.FullName, &u.AvatarURL, &u.FollowerCount, &u.FollowingCount, &u.IsFollowing); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

func (r *SocialRepository) GetFollowing(ctx context.Context, userID uuid.UUID, requestingUserID uuid.UUID, page, limit int) ([]models.UserSearchResult, int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_follows WHERE follower_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	query := `
		SELECT u.id, u.full_name, u.avatar_url, u.follower_count, u.following_count,
			EXISTS(SELECT 1 FROM user_follows WHERE follower_id = $2 AND following_id = u.id) as is_following
		FROM user_follows uf
		JOIN users u ON uf.following_id = u.id
		WHERE uf.follower_id = $1
		ORDER BY uf.created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.pool.Query(ctx, query, userID, requestingUserID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.UserSearchResult
	for rows.Next() {
		var u models.UserSearchResult
		if err := rows.Scan(&u.ID, &u.FullName, &u.AvatarURL, &u.FollowerCount, &u.FollowingCount, &u.IsFollowing); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, nil
}

func (r *SocialRepository) SearchUsers(ctx context.Context, q string, requestingUserID uuid.UUID, limit int) ([]models.UserSearchResult, error) {
	query := `
		SELECT u.id, u.full_name, u.avatar_url, u.follower_count, u.following_count,
			EXISTS(SELECT 1 FROM user_follows WHERE follower_id = $2 AND following_id = u.id) as is_following
		FROM users u
		WHERE u.full_name ILIKE '%' || $1 || '%' AND u.id != $2 AND u.is_active = true
		ORDER BY u.full_name
		LIMIT $3`

	rows, err := r.pool.Query(ctx, query, q, requestingUserID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.UserSearchResult
	for rows.Next() {
		var u models.UserSearchResult
		if err := rows.Scan(&u.ID, &u.FullName, &u.AvatarURL, &u.FollowerCount, &u.FollowingCount, &u.IsFollowing); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (r *SocialRepository) GetFollowedUserIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	query := `SELECT following_id FROM user_follows WHERE follower_id = $1`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
