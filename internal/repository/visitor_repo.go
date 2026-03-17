package repository

import (
	"apartment-backend/internal/models"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VisitorRepository struct {
	pool *pgxpool.Pool
}

func NewVisitorRepository(pool *pgxpool.Pool) *VisitorRepository {
	return &VisitorRepository{pool: pool}
}

func (r *VisitorRepository) Create(ctx context.Context, buildingID, userID uuid.UUID, req *models.CreateVisitorRequest) (*models.VisitorPass, error) {
	qrCode := uuid.New().String()

	var pass models.VisitorPass
	query := `
		INSERT INTO visitor_passes (building_id, created_by, visitor_name, visitor_phone, visitor_plate, purpose, expected_at, qr_code, notes, status)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, '')::timestamptz, $8, $9, 'pending')
		RETURNING id, building_id, created_by, visitor_name, visitor_phone, visitor_plate, purpose, expected_at, qr_code, status, notes, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		buildingID, userID, req.VisitorName, req.VisitorPhone, req.VisitorPlate, req.Purpose, req.ExpectedAt, qrCode, req.Notes,
	).Scan(&pass.ID, &pass.BuildingID, &pass.CreatedBy, &pass.VisitorName, &pass.VisitorPhone, &pass.VisitorPlate, &pass.Purpose, &pass.ExpectedAt, &pass.QRCode, &pass.Status, &pass.Notes, &pass.CreatedAt, &pass.UpdatedAt)

	return &pass, err
}

func (r *VisitorRepository) GetByBuilding(ctx context.Context, buildingID uuid.UUID, status string, page, limit int) ([]models.VisitorPass, error) {
	offset := (page - 1) * limit
	whereStatus := ""
	args := []interface{}{buildingID, limit, offset}

	if status != "" && status != "all" {
		whereStatus = " AND vp.status = $4"
		args = append(args, status)
	}

	query := fmt.Sprintf(`
		SELECT vp.id, vp.building_id, vp.created_by, vp.visitor_name, vp.visitor_phone, vp.visitor_plate,
			vp.purpose, vp.expected_at, vp.checked_in_at, vp.checked_out_at, vp.qr_code, vp.status,
			vp.notes, vp.created_at, vp.updated_at,
			COALESCE(u.full_name, '') as created_by_name
		FROM visitor_passes vp
		LEFT JOIN users u ON u.id = vp.created_by
		WHERE vp.building_id = $1%s
		ORDER BY vp.created_at DESC
		LIMIT $2 OFFSET $3`, whereStatus)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var passes []models.VisitorPass
	for rows.Next() {
		var p models.VisitorPass
		err := rows.Scan(
			&p.ID, &p.BuildingID, &p.CreatedBy, &p.VisitorName, &p.VisitorPhone, &p.VisitorPlate,
			&p.Purpose, &p.ExpectedAt, &p.CheckedInAt, &p.CheckedOutAt, &p.QRCode, &p.Status,
			&p.Notes, &p.CreatedAt, &p.UpdatedAt, &p.CreatedByName,
		)
		if err != nil {
			return nil, err
		}
		passes = append(passes, p)
	}
	return passes, nil
}

func (r *VisitorRepository) CheckIn(ctx context.Context, id uuid.UUID, approvedBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE visitor_passes SET status = 'checked_in', checked_in_at = NOW(), approved_by = $2, updated_at = NOW() WHERE id = $1`,
		id, approvedBy)
	return err
}

func (r *VisitorRepository) CheckOut(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE visitor_passes SET status = 'checked_out', checked_out_at = NOW(), updated_at = NOW() WHERE id = $1`,
		id)
	return err
}

func (r *VisitorRepository) Cancel(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE visitor_passes SET status = 'cancelled', updated_at = NOW() WHERE id = $1`,
		id)
	return err
}

func (r *VisitorRepository) GetByQR(ctx context.Context, qrCode string) (*models.VisitorPass, error) {
	var p models.VisitorPass
	err := r.pool.QueryRow(ctx,
		`SELECT id, building_id, created_by, visitor_name, visitor_phone, visitor_plate, purpose, expected_at,
			checked_in_at, checked_out_at, qr_code, status, notes, created_at, updated_at
		FROM visitor_passes WHERE qr_code = $1`, qrCode,
	).Scan(&p.ID, &p.BuildingID, &p.CreatedBy, &p.VisitorName, &p.VisitorPhone, &p.VisitorPlate, &p.Purpose, &p.ExpectedAt,
		&p.CheckedInAt, &p.CheckedOutAt, &p.QRCode, &p.Status, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	return &p, err
}
