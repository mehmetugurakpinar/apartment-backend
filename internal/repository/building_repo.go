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

type BuildingRepository struct {
	pool *pgxpool.Pool
}

func NewBuildingRepository(pool *pgxpool.Pool) *BuildingRepository {
	return &BuildingRepository{pool: pool}
}

func (r *BuildingRepository) Create(ctx context.Context, b *models.Building) error {
	query := `
		INSERT INTO buildings (name, address, city, total_units, latitude, longitude, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		b.Name, b.Address, b.City, b.TotalUnits, b.Latitude, b.Longitude, b.CreatedBy,
	).Scan(&b.ID, &b.CreatedAt, &b.UpdatedAt)
}

func (r *BuildingRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Building, error) {
	query := `
		SELECT id, name, address, city, total_units, latitude, longitude, image_url, created_by, created_at, updated_at
		FROM buildings WHERE id = $1`

	b := &models.Building{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&b.ID, &b.Name, &b.Address, &b.City, &b.TotalUnits,
		&b.Latitude, &b.Longitude, &b.ImageURL, &b.CreatedBy,
		&b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("building not found")
		}
		return nil, err
	}
	return b, nil
}

func (r *BuildingRepository) GetDashboard(ctx context.Context, buildingID uuid.UUID) (*models.BuildingDashboard, error) {
	building, err := r.GetByID(ctx, buildingID)
	if err != nil {
		return nil, err
	}

	dash := &models.BuildingDashboard{Building: *building}

	// Unit counts
	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM units WHERE building_id = $1`, buildingID).Scan(&dash.TotalUnits)
	if err != nil {
		return nil, err
	}

	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM units WHERE building_id = $1 AND status = 'occupied'`, buildingID).Scan(&dash.OccupiedUnits)
	if err != nil {
		return nil, err
	}
	dash.VacantUnits = dash.TotalUnits - dash.OccupiedUnits

	// Resident count
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT ur.user_id) FROM unit_residents ur
		JOIN units u ON ur.unit_id = u.id
		WHERE u.building_id = $1 AND ur.is_active = true
	`, buildingID).Scan(&dash.TotalResidents)
	if err != nil {
		return nil, err
	}

	// Pending dues
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM due_payments dp
		JOIN dues_plans p ON dp.dues_plan_id = p.id
		WHERE p.building_id = $1 AND dp.status = 'pending'
	`, buildingID).Scan(&dash.PendingDues)
	if err != nil {
		return nil, err
	}

	// Open maintenance
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM maintenance_requests
		WHERE building_id = $1 AND status IN ('open', 'in_progress')
	`, buildingID).Scan(&dash.OpenRequests)
	if err != nil {
		return nil, err
	}

	return dash, nil
}

func (r *BuildingRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]models.Building, error) {
	query := `
		SELECT b.id, b.name, b.address, b.city, b.total_units, b.latitude, b.longitude, b.created_by, b.created_at, b.updated_at
		FROM buildings b
		INNER JOIN building_members bm ON b.id = bm.building_id
		WHERE bm.user_id = $1
		ORDER BY b.created_at DESC`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buildings []models.Building
	for rows.Next() {
		var b models.Building
		if err := rows.Scan(&b.ID, &b.Name, &b.Address, &b.City, &b.TotalUnits, &b.Latitude, &b.Longitude, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		buildings = append(buildings, b)
	}
	return buildings, nil
}

func (r *BuildingRepository) AddMember(ctx context.Context, member *models.BuildingMember) error {
	query := `
		INSERT INTO building_members (building_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (building_id, user_id) DO UPDATE SET role = $3
		RETURNING id, joined_at`
	return r.pool.QueryRow(ctx, query, member.BuildingID, member.UserID, member.Role).Scan(&member.ID, &member.JoinedAt)
}

func (r *BuildingRepository) IsMember(ctx context.Context, buildingID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM building_members WHERE building_id = $1 AND user_id = $2)`,
		buildingID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *BuildingRepository) GetResidents(ctx context.Context, buildingID uuid.UUID) ([]models.UserResponse, error) {
	query := `
		SELECT u.id, u.email, u.full_name, u.phone, u.avatar_url, u.role, u.created_at
		FROM users u
		JOIN building_members bm ON u.id = bm.user_id
		WHERE bm.building_id = $1
		ORDER BY u.full_name`

	rows, err := r.pool.Query(ctx, query, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var residents []models.UserResponse
	for rows.Next() {
		var u models.UserResponse
		if err := rows.Scan(&u.ID, &u.Email, &u.FullName, &u.Phone, &u.AvatarURL, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		residents = append(residents, u)
	}
	return residents, nil
}

// Units

func (r *BuildingRepository) CreateUnit(ctx context.Context, unit *models.Unit) error {
	query := `
		INSERT INTO units (building_id, block, floor, unit_number, area_sqm, owner_id, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		unit.BuildingID, unit.Block, unit.Floor, unit.UnitNumber, unit.AreaSqm, unit.OwnerID, unit.Status,
	).Scan(&unit.ID, &unit.CreatedAt, &unit.UpdatedAt)
}

func (r *BuildingRepository) GetUnits(ctx context.Context, buildingID uuid.UUID) ([]models.Unit, error) {
	query := `
		SELECT id, building_id, block, floor, unit_number, area_sqm, owner_id, status, created_at, updated_at
		FROM units WHERE building_id = $1
		ORDER BY block, floor, unit_number`

	rows, err := r.pool.Query(ctx, query, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	units := make([]models.Unit, 0)
	for rows.Next() {
		var u models.Unit
		if err := rows.Scan(
			&u.ID, &u.BuildingID, &u.Block, &u.Floor, &u.UnitNumber,
			&u.AreaSqm, &u.OwnerID, &u.Status, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		units = append(units, u)
	}
	return units, nil
}

func (r *BuildingRepository) GetUnitByID(ctx context.Context, unitID uuid.UUID) (*models.Unit, error) {
	query := `
		SELECT id, building_id, block, floor, unit_number, area_sqm, owner_id, status, created_at, updated_at
		FROM units WHERE id = $1`

	u := &models.Unit{}
	err := r.pool.QueryRow(ctx, query, unitID).Scan(
		&u.ID, &u.BuildingID, &u.Block, &u.Floor, &u.UnitNumber,
		&u.AreaSqm, &u.OwnerID, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("unit not found")
		}
		return nil, err
	}
	return u, nil
}

func (r *BuildingRepository) UpdateUnit(ctx context.Context, unitID uuid.UUID, req *models.UpdateUnitRequest) (*models.Unit, error) {
	query := `
		UPDATE units SET
			block = COALESCE($2, block),
			floor = COALESCE($3, floor),
			unit_number = COALESCE($4, unit_number),
			area_sqm = COALESCE($5, area_sqm),
			owner_id = COALESCE($6, owner_id),
			status = COALESCE($7, status)
		WHERE id = $1
		RETURNING id, building_id, block, floor, unit_number, area_sqm, owner_id, status, created_at, updated_at`

	var ownerID *uuid.UUID
	if req.OwnerID != nil {
		id, err := uuid.Parse(*req.OwnerID)
		if err == nil {
			ownerID = &id
		}
	}

	u := &models.Unit{}
	err := r.pool.QueryRow(ctx, query,
		unitID, req.Block, req.Floor, req.UnitNumber, req.AreaSqm, ownerID, req.Status,
	).Scan(
		&u.ID, &u.BuildingID, &u.Block, &u.Floor, &u.UnitNumber,
		&u.AreaSqm, &u.OwnerID, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *BuildingRepository) DeleteUnit(ctx context.Context, unitID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM units WHERE id = $1`, unitID)
	return err
}

func (r *BuildingRepository) GetMemberRole(ctx context.Context, buildingID, userID uuid.UUID) (models.UserRole, error) {
	var role models.UserRole
	err := r.pool.QueryRow(ctx,
		`SELECT role FROM building_members WHERE building_id = $1 AND user_id = $2`,
		buildingID, userID,
	).Scan(&role)
	if err != nil {
		return "", err
	}
	return role, nil
}

func (r *BuildingRepository) RemoveMember(ctx context.Context, buildingID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM building_members WHERE building_id = $1 AND user_id = $2`,
		buildingID, userID,
	)
	return err
}

func (r *BuildingRepository) GetMembers(ctx context.Context, buildingID uuid.UUID) ([]models.BuildingMemberDetail, error) {
	query := `
		SELECT u.id, u.email, u.full_name, u.phone, u.avatar_url, bm.role, bm.joined_at
		FROM users u
		JOIN building_members bm ON u.id = bm.user_id
		WHERE bm.building_id = $1
		ORDER BY bm.role, u.full_name`

	rows, err := r.pool.Query(ctx, query, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]models.BuildingMemberDetail, 0)
	for rows.Next() {
		var m models.BuildingMemberDetail
		if err := rows.Scan(&m.ID, &m.Email, &m.FullName, &m.Phone, &m.AvatarURL, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

// Invitations

func (r *BuildingRepository) CreateInvitation(ctx context.Context, inv *models.BuildingInvitation) error {
	query := `
		INSERT INTO building_invitations (building_id, email, role, token, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		inv.BuildingID, inv.Email, inv.Role, inv.Token, inv.InvitedBy, inv.ExpiresAt,
	).Scan(&inv.ID, &inv.CreatedAt)
}

func (r *BuildingRepository) GetInvitationByToken(ctx context.Context, token string) (*models.BuildingInvitation, error) {
	query := `
		SELECT id, building_id, email, role, token, invited_by, accepted_at, expires_at, created_at
		FROM building_invitations WHERE token = $1`

	inv := &models.BuildingInvitation{}
	err := r.pool.QueryRow(ctx, query, token).Scan(
		&inv.ID, &inv.BuildingID, &inv.Email, &inv.Role, &inv.Token,
		&inv.InvitedBy, &inv.AcceptedAt, &inv.ExpiresAt, &inv.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("invitation not found")
		}
		return nil, err
	}
	return inv, nil
}

func (r *BuildingRepository) GetInvitationsByBuilding(ctx context.Context, buildingID uuid.UUID) ([]models.BuildingInvitation, error) {
	query := `
		SELECT id, building_id, email, role, token, invited_by, accepted_at, expires_at, created_at
		FROM building_invitations WHERE building_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []models.BuildingInvitation
	for rows.Next() {
		var inv models.BuildingInvitation
		if err := rows.Scan(
			&inv.ID, &inv.BuildingID, &inv.Email, &inv.Role, &inv.Token,
			&inv.InvitedBy, &inv.AcceptedAt, &inv.ExpiresAt, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		invitations = append(invitations, inv)
	}
	return invitations, nil
}

func (r *BuildingRepository) MarkInvitationAccepted(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE building_invitations SET accepted_at = $2 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, time.Now())
	return err
}
