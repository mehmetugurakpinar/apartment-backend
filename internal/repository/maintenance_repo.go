package repository

import (
	"context"
	"fmt"
	"time"

	"apartment-backend/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MaintenanceRepository struct {
	pool *pgxpool.Pool
}

func NewMaintenanceRepository(pool *pgxpool.Pool) *MaintenanceRepository {
	return &MaintenanceRepository{pool: pool}
}

func (r *MaintenanceRepository) Create(ctx context.Context, req *models.MaintenanceRequest) error {
	query := `
		INSERT INTO maintenance_requests (building_id, unit_id, title, description, priority, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, status, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		req.BuildingID, req.UnitID, req.Title, req.Description, req.Priority, req.CreatedBy,
	).Scan(&req.ID, &req.Status, &req.CreatedAt, &req.UpdatedAt)
}

func (r *MaintenanceRepository) GetByBuilding(ctx context.Context, buildingID uuid.UUID, page, limit int) ([]models.MaintenanceRequestDetail, int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM maintenance_requests WHERE building_id = $1`, buildingID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT mr.id, mr.building_id, mr.unit_id, mr.title, mr.description, mr.priority,
			mr.status, mr.assigned_to, mr.created_by, mr.created_at, mr.updated_at, mr.resolved_at,
			u_creator.full_name as created_by_name,
			u_assigned.full_name as assigned_to_name,
			unit.unit_number
		FROM maintenance_requests mr
		JOIN users u_creator ON mr.created_by = u_creator.id
		LEFT JOIN users u_assigned ON mr.assigned_to = u_assigned.id
		LEFT JOIN units unit ON mr.unit_id = unit.id
		WHERE mr.building_id = $1
		ORDER BY
			CASE mr.priority WHEN 'emergency' THEN 0 WHEN 'high' THEN 1 WHEN 'normal' THEN 2 ELSE 3 END,
			mr.created_at DESC
		LIMIT $2 OFFSET $3`

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx, query, buildingID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var requests []models.MaintenanceRequestDetail
	for rows.Next() {
		var d models.MaintenanceRequestDetail
		if err := rows.Scan(
			&d.ID, &d.BuildingID, &d.UnitID, &d.Title, &d.Description,
			&d.Priority, &d.Status, &d.AssignedTo, &d.CreatedBy,
			&d.CreatedAt, &d.UpdatedAt, &d.ResolvedAt,
			&d.CreatedByName, &d.AssignedToName, &d.UnitNumber,
		); err != nil {
			return nil, 0, err
		}
		requests = append(requests, d)
	}
	return requests, total, nil
}

func (r *MaintenanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.MaintenanceRequestDetail, error) {
	query := `
		SELECT mr.id, mr.building_id, mr.unit_id, mr.title, mr.description, mr.priority,
			mr.status, mr.assigned_to, mr.created_by, mr.created_at, mr.updated_at, mr.resolved_at,
			u_creator.full_name as created_by_name,
			u_assigned.full_name as assigned_to_name,
			unit.unit_number
		FROM maintenance_requests mr
		JOIN users u_creator ON mr.created_by = u_creator.id
		LEFT JOIN users u_assigned ON mr.assigned_to = u_assigned.id
		LEFT JOIN units unit ON mr.unit_id = unit.id
		WHERE mr.id = $1`

	d := &models.MaintenanceRequestDetail{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.BuildingID, &d.UnitID, &d.Title, &d.Description,
		&d.Priority, &d.Status, &d.AssignedTo, &d.CreatedBy,
		&d.CreatedAt, &d.UpdatedAt, &d.ResolvedAt,
		&d.CreatedByName, &d.AssignedToName, &d.UnitNumber,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("maintenance request not found")
		}
		return nil, err
	}

	// Get photos
	photos, err := r.GetPhotos(ctx, id)
	if err != nil {
		return nil, err
	}
	d.Photos = photos

	return d, nil
}

func (r *MaintenanceRepository) Update(ctx context.Context, id uuid.UUID, req *models.UpdateMaintenanceRequest) error {
	var assignedTo *uuid.UUID
	if req.AssignedTo != nil {
		parsed, err := uuid.Parse(*req.AssignedTo)
		if err == nil {
			assignedTo = &parsed
		}
	}

	var resolvedAt *time.Time
	if req.Status != nil && (*req.Status == models.MaintenanceResolved || *req.Status == models.MaintenanceClosed) {
		now := time.Now()
		resolvedAt = &now
	}

	query := `
		UPDATE maintenance_requests SET
			status = COALESCE($2, status),
			priority = COALESCE($3, priority),
			assigned_to = COALESCE($4, assigned_to),
			resolved_at = COALESCE($5, resolved_at)
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, query, id, req.Status, req.Priority, assignedTo, resolvedAt)
	return err
}

func (r *MaintenanceRepository) AddPhoto(ctx context.Context, photo *models.MaintenancePhoto) error {
	query := `INSERT INTO maintenance_photos (request_id, url) VALUES ($1, $2) RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query, photo.RequestID, photo.URL).Scan(&photo.ID, &photo.CreatedAt)
}

func (r *MaintenanceRepository) GetPhotos(ctx context.Context, requestID uuid.UUID) ([]models.MaintenancePhoto, error) {
	query := `SELECT id, request_id, url, created_at FROM maintenance_photos WHERE request_id = $1`
	rows, err := r.pool.Query(ctx, query, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var photos []models.MaintenancePhoto
	for rows.Next() {
		var p models.MaintenancePhoto
		if err := rows.Scan(&p.ID, &p.RequestID, &p.URL, &p.CreatedAt); err != nil {
			return nil, err
		}
		photos = append(photos, p)
	}
	return photos, nil
}

// Vendors

func (r *MaintenanceRepository) CreateVendor(ctx context.Context, vendor *models.Vendor) error {
	query := `
		INSERT INTO vendors (building_id, name, category, phone, email)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query,
		vendor.BuildingID, vendor.Name, vendor.Category, vendor.Phone, vendor.Email,
	).Scan(&vendor.ID, &vendor.CreatedAt)
}

func (r *MaintenanceRepository) DeleteRequest(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM maintenance_requests WHERE id = $1`, id)
	return err
}

func (r *MaintenanceRepository) UpdateVendor(ctx context.Context, id uuid.UUID, req *models.UpdateVendorRequest) (*models.Vendor, error) {
	query := `
		UPDATE vendors SET
			name = COALESCE($2, name),
			category = COALESCE($3, category),
			phone = COALESCE($4, phone),
			email = COALESCE($5, email)
		WHERE id = $1
		RETURNING id, building_id, name, category, phone, email, rating, created_at`

	v := &models.Vendor{}
	err := r.pool.QueryRow(ctx, query, id, req.Name, req.Category, req.Phone, req.Email).Scan(
		&v.ID, &v.BuildingID, &v.Name, &v.Category, &v.Phone, &v.Email, &v.Rating, &v.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *MaintenanceRepository) DeleteVendor(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM vendors WHERE id = $1`, id)
	return err
}

func (r *MaintenanceRepository) GetVendors(ctx context.Context, buildingID uuid.UUID) ([]models.Vendor, error) {
	query := `
		SELECT id, building_id, name, category, phone, email, rating, created_at
		FROM vendors WHERE building_id = $1
		ORDER BY name`

	rows, err := r.pool.Query(ctx, query, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vendors := make([]models.Vendor, 0)
	for rows.Next() {
		var v models.Vendor
		if err := rows.Scan(&v.ID, &v.BuildingID, &v.Name, &v.Category, &v.Phone, &v.Email, &v.Rating, &v.CreatedAt); err != nil {
			return nil, err
		}
		vendors = append(vendors, v)
	}
	return vendors, nil
}
