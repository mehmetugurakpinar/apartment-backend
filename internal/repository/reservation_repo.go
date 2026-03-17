package repository

import (
	"apartment-backend/internal/models"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReservationRepository struct {
	pool *pgxpool.Pool
}

func NewReservationRepository(pool *pgxpool.Pool) *ReservationRepository {
	return &ReservationRepository{pool: pool}
}

// Common Areas

func (r *ReservationRepository) CreateArea(ctx context.Context, buildingID uuid.UUID, req *models.CreateCommonAreaRequest) (*models.CommonArea, error) {
	var area models.CommonArea
	openTime := "08:00"
	closeTime := "22:00"
	if req.OpenTime != "" {
		openTime = req.OpenTime
	}
	if req.CloseTime != "" {
		closeTime = req.CloseTime
	}

	err := r.pool.QueryRow(ctx,
		`INSERT INTO common_areas (building_id, name, description, capacity, rules, open_time, close_time, requires_approval)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, building_id, name, description, capacity, rules, open_time::text, close_time::text, requires_approval, is_active, image_url, created_at, updated_at`,
		buildingID, req.Name, req.Description, req.Capacity, req.Rules, openTime, closeTime, req.RequiresApproval,
	).Scan(&area.ID, &area.BuildingID, &area.Name, &area.Description, &area.Capacity, &area.Rules,
		&area.OpenTime, &area.CloseTime, &area.RequiresApproval, &area.IsActive, &area.ImageURL, &area.CreatedAt, &area.UpdatedAt)
	return &area, err
}

func (r *ReservationRepository) GetAreas(ctx context.Context, buildingID uuid.UUID) ([]models.CommonArea, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, building_id, name, description, capacity, rules, open_time::text, close_time::text,
			requires_approval, is_active, COALESCE(image_url, ''), created_at, updated_at
		FROM common_areas WHERE building_id = $1 AND is_active = true ORDER BY name`, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var areas []models.CommonArea
	for rows.Next() {
		var a models.CommonArea
		err := rows.Scan(&a.ID, &a.BuildingID, &a.Name, &a.Description, &a.Capacity, &a.Rules,
			&a.OpenTime, &a.CloseTime, &a.RequiresApproval, &a.IsActive, &a.ImageURL, &a.CreatedAt, &a.UpdatedAt)
		if err != nil {
			return nil, err
		}
		areas = append(areas, a)
	}
	return areas, nil
}

// Reservations

func (r *ReservationRepository) CreateReservation(ctx context.Context, buildingID, userID uuid.UUID, req *models.CreateReservationRequest) (*models.Reservation, error) {
	areaID, err := uuid.Parse(req.CommonAreaID)
	if err != nil {
		return nil, err
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		return nil, err
	}
	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		return nil, err
	}

	guestCount := 1
	if req.GuestCount > 0 {
		guestCount = req.GuestCount
	}

	// Check if area requires approval
	var requiresApproval bool
	err = r.pool.QueryRow(ctx, `SELECT requires_approval FROM common_areas WHERE id = $1`, areaID).Scan(&requiresApproval)
	if err != nil {
		return nil, err
	}

	status := "approved"
	if requiresApproval {
		status = "pending"
	}

	var res models.Reservation
	err = r.pool.QueryRow(ctx,
		`INSERT INTO reservations (common_area_id, user_id, building_id, title, start_time, end_time, guest_count, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, common_area_id, user_id, building_id, title, start_time, end_time, guest_count, status, notes, created_at, updated_at`,
		areaID, userID, buildingID, req.Title, startTime, endTime, guestCount, status, req.Notes,
	).Scan(&res.ID, &res.CommonAreaID, &res.UserID, &res.BuildingID, &res.Title, &res.StartTime, &res.EndTime,
		&res.GuestCount, &res.Status, &res.Notes, &res.CreatedAt, &res.UpdatedAt)
	return &res, err
}

func (r *ReservationRepository) GetReservations(ctx context.Context, buildingID uuid.UUID, areaID *uuid.UUID, page, limit int) ([]models.Reservation, error) {
	offset := (page - 1) * limit
	args := []interface{}{buildingID, limit, offset}

	areaFilter := ""
	if areaID != nil {
		areaFilter = " AND r.common_area_id = $4"
		args = append(args, *areaID)
	}

	query := `
		SELECT r.id, r.common_area_id, r.user_id, r.building_id, r.title, r.start_time, r.end_time,
			r.guest_count, r.status, r.notes, r.created_at, r.updated_at,
			COALESCE(ca.name, '') as area_name, COALESCE(u.full_name, '') as user_name
		FROM reservations r
		LEFT JOIN common_areas ca ON ca.id = r.common_area_id
		LEFT JOIN users u ON u.id = r.user_id
		WHERE r.building_id = $1` + areaFilter + `
		ORDER BY r.start_time DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []models.Reservation
	for rows.Next() {
		var res models.Reservation
		err := rows.Scan(&res.ID, &res.CommonAreaID, &res.UserID, &res.BuildingID, &res.Title,
			&res.StartTime, &res.EndTime, &res.GuestCount, &res.Status, &res.Notes,
			&res.CreatedAt, &res.UpdatedAt, &res.AreaName, &res.UserName)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, res)
	}
	return reservations, nil
}

func (r *ReservationRepository) UpdateReservationStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE reservations SET status = $2, updated_at = NOW() WHERE id = $1`, id, status)
	return err
}

func (r *ReservationRepository) GetMyReservations(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Reservation, error) {
	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.common_area_id, r.user_id, r.building_id, r.title, r.start_time, r.end_time,
			r.guest_count, r.status, r.notes, r.created_at, r.updated_at,
			COALESCE(ca.name, '') as area_name, COALESCE(u.full_name, '') as user_name
		FROM reservations r
		LEFT JOIN common_areas ca ON ca.id = r.common_area_id
		LEFT JOIN users u ON u.id = r.user_id
		WHERE r.user_id = $1
		ORDER BY r.start_time DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reservations []models.Reservation
	for rows.Next() {
		var res models.Reservation
		err := rows.Scan(&res.ID, &res.CommonAreaID, &res.UserID, &res.BuildingID, &res.Title,
			&res.StartTime, &res.EndTime, &res.GuestCount, &res.Status, &res.Notes,
			&res.CreatedAt, &res.UpdatedAt, &res.AreaName, &res.UserName)
		if err != nil {
			return nil, err
		}
		reservations = append(reservations, res)
	}
	return reservations, nil
}
