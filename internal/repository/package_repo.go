package repository

import (
	"apartment-backend/internal/models"
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PackageRepository struct {
	pool *pgxpool.Pool
}

func NewPackageRepository(pool *pgxpool.Pool) *PackageRepository {
	return &PackageRepository{pool: pool}
}

func (r *PackageRepository) Create(ctx context.Context, buildingID, receivedBy uuid.UUID, req *models.CreatePackageRequest) (*models.Package, error) {
	var pkg models.Package
	var recipientID *uuid.UUID
	if req.RecipientID != "" {
		id, err := uuid.Parse(req.RecipientID)
		if err == nil {
			recipientID = &id
		}
	}

	err := r.pool.QueryRow(ctx,
		`INSERT INTO packages (building_id, recipient_id, carrier, tracking_number, description, received_by, notes, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'waiting')
		RETURNING id, building_id, recipient_id, carrier, tracking_number, description, received_by, received_at, status, notes, created_at, updated_at`,
		buildingID, recipientID, req.Carrier, req.TrackingNumber, req.Description, receivedBy, req.Notes,
	).Scan(&pkg.ID, &pkg.BuildingID, &pkg.RecipientID, &pkg.Carrier, &pkg.TrackingNumber, &pkg.Description,
		&pkg.ReceivedBy, &pkg.ReceivedAt, &pkg.Status, &pkg.Notes, &pkg.CreatedAt, &pkg.UpdatedAt)
	return &pkg, err
}

func (r *PackageRepository) GetByBuilding(ctx context.Context, buildingID uuid.UUID, status string, page, limit int) ([]models.Package, error) {
	offset := (page - 1) * limit
	args := []interface{}{buildingID, limit, offset}

	statusFilter := ""
	if status != "" && status != "all" {
		statusFilter = " AND p.status = $4"
		args = append(args, status)
	}

	query := `
		SELECT p.id, p.building_id, p.recipient_id, p.carrier, p.tracking_number, p.description,
			p.received_by, p.received_at, p.picked_up_by, p.picked_up_at, COALESCE(p.photo_url, ''),
			p.status, p.notes, p.created_at, p.updated_at,
			COALESCE(u.full_name, '') as recipient_name,
			COALESCE(rb.full_name, '') as received_by_name
		FROM packages p
		LEFT JOIN users u ON u.id = p.recipient_id
		LEFT JOIN users rb ON rb.id = p.received_by
		WHERE p.building_id = $1` + statusFilter + `
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []models.Package
	for rows.Next() {
		var pkg models.Package
		err := rows.Scan(
			&pkg.ID, &pkg.BuildingID, &pkg.RecipientID, &pkg.Carrier, &pkg.TrackingNumber, &pkg.Description,
			&pkg.ReceivedBy, &pkg.ReceivedAt, &pkg.PickedUpBy, &pkg.PickedUpAt, &pkg.PhotoURL,
			&pkg.Status, &pkg.Notes, &pkg.CreatedAt, &pkg.UpdatedAt,
			&pkg.RecipientName, &pkg.ReceivedByName,
		)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}

func (r *PackageRepository) MarkPickedUp(ctx context.Context, id, pickedUpBy uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE packages SET status = 'picked_up', picked_up_by = $2, picked_up_at = NOW(), updated_at = NOW() WHERE id = $1`,
		id, pickedUpBy)
	return err
}

func (r *PackageRepository) NotifyRecipient(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE packages SET status = 'notified', updated_at = NOW() WHERE id = $1`, id)
	return err
}

func (r *PackageRepository) GetMyPackages(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Package, error) {
	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.building_id, p.recipient_id, p.carrier, p.tracking_number, p.description,
			p.received_by, p.received_at, p.picked_up_by, p.picked_up_at, COALESCE(p.photo_url, ''),
			p.status, p.notes, p.created_at, p.updated_at,
			'' as recipient_name,
			COALESCE(rb.full_name, '') as received_by_name
		FROM packages p
		LEFT JOIN users rb ON rb.id = p.received_by
		WHERE p.recipient_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var packages []models.Package
	for rows.Next() {
		var pkg models.Package
		err := rows.Scan(
			&pkg.ID, &pkg.BuildingID, &pkg.RecipientID, &pkg.Carrier, &pkg.TrackingNumber, &pkg.Description,
			&pkg.ReceivedBy, &pkg.ReceivedAt, &pkg.PickedUpBy, &pkg.PickedUpAt, &pkg.PhotoURL,
			&pkg.Status, &pkg.Notes, &pkg.CreatedAt, &pkg.UpdatedAt,
			&pkg.RecipientName, &pkg.ReceivedByName,
		)
		if err != nil {
			return nil, err
		}
		packages = append(packages, pkg)
	}
	return packages, nil
}
