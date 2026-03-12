package repository

import (
	"context"
	"encoding/json"

	"apartment-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) Create(ctx context.Context, n *models.Notification) error {
	query := `
		INSERT INTO notifications (user_id, building_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query,
		n.UserID, n.BuildingID, n.Type, n.Title, n.Body, n.Data,
	).Scan(&n.ID, &n.CreatedAt)
}

func (r *NotificationRepository) CreateBulk(ctx context.Context, buildingID uuid.UUID, notifType, title, body string, data json.RawMessage) error {
	query := `
		INSERT INTO notifications (user_id, building_id, type, title, body, data)
		SELECT bm.user_id, $1, $2, $3, $4, $5
		FROM building_members bm
		WHERE bm.building_id = $1`
	_, err := r.pool.Exec(ctx, query, buildingID, notifType, title, body, data)
	return err
}

func (r *NotificationRepository) GetByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Notification, int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM notifications WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, user_id, building_id, type, title, body, data, read_at, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.BuildingID, &n.Type,
			&n.Title, &n.Body, &n.Data, &n.ReadAt, &n.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		notifications = append(notifications, n)
	}
	return notifications, total, nil
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	query := `UPDATE notifications SET read_at = NOW() WHERE id = $1 AND user_id = $2`
	_, err := r.pool.Exec(ctx, query, id, userID)
	return err
}

func (r *NotificationRepository) GetPreferences(ctx context.Context, userID uuid.UUID) ([]models.NotificationPreference, error) {
	query := `
		SELECT id, user_id, category, push_enabled, email_enabled
		FROM notification_preferences WHERE user_id = $1`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []models.NotificationPreference
	for rows.Next() {
		var p models.NotificationPreference
		if err := rows.Scan(&p.ID, &p.UserID, &p.Category, &p.PushEnabled, &p.EmailEnabled); err != nil {
			return nil, err
		}
		prefs = append(prefs, p)
	}
	return prefs, nil
}

func (r *NotificationRepository) UpsertPreference(ctx context.Context, userID uuid.UUID, req *models.UpdatePreferencesRequest) error {
	query := `
		INSERT INTO notification_preferences (user_id, category, push_enabled, email_enabled)
		VALUES ($1, $2, COALESCE($3, true), COALESCE($4, false))
		ON CONFLICT (user_id, category) DO UPDATE SET
			push_enabled = COALESCE($3, notification_preferences.push_enabled),
			email_enabled = COALESCE($4, notification_preferences.email_enabled)`

	_, err := r.pool.Exec(ctx, query, userID, req.Category, req.PushEnabled, req.EmailEnabled)
	return err
}

func (r *NotificationRepository) GetBuildingUserFCMTokens(ctx context.Context, buildingID uuid.UUID) ([]string, error) {
	query := `
		SELECT u.fcm_token FROM users u
		JOIN building_members bm ON u.id = bm.user_id
		WHERE bm.building_id = $1 AND u.fcm_token IS NOT NULL AND u.fcm_token != ''`

	rows, err := r.pool.Query(ctx, query, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, nil
}
