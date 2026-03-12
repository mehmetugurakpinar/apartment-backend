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

type FinancialRepository struct {
	pool *pgxpool.Pool
}

func NewFinancialRepository(pool *pgxpool.Pool) *FinancialRepository {
	return &FinancialRepository{pool: pool}
}

// Dues Plans

func (r *FinancialRepository) CreateDuesPlan(ctx context.Context, plan *models.DuesPlan) error {
	query := `
		INSERT INTO dues_plans (building_id, title, amount, period_month, period_year, due_date, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query,
		plan.BuildingID, plan.Title, plan.Amount, plan.PeriodMonth, plan.PeriodYear, plan.DueDate, plan.CreatedBy,
	).Scan(&plan.ID, &plan.CreatedAt)
}

func (r *FinancialRepository) GetDuesPlans(ctx context.Context, buildingID uuid.UUID) ([]models.DuesPlan, error) {
	query := `
		SELECT id, building_id, title, amount, period_month, period_year, due_date, created_by, created_at
		FROM dues_plans WHERE building_id = $1
		ORDER BY period_year DESC, period_month DESC`

	rows, err := r.pool.Query(ctx, query, buildingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []models.DuesPlan
	for rows.Next() {
		var p models.DuesPlan
		if err := rows.Scan(
			&p.ID, &p.BuildingID, &p.Title, &p.Amount,
			&p.PeriodMonth, &p.PeriodYear, &p.DueDate, &p.CreatedBy, &p.CreatedAt,
		); err != nil {
			return nil, err
		}
		plans = append(plans, p)
	}
	return plans, nil
}

func (r *FinancialRepository) GetDuesPlanByID(ctx context.Context, planID uuid.UUID) (*models.DuesPlan, error) {
	query := `
		SELECT id, building_id, title, amount, period_month, period_year, due_date, created_by, created_at
		FROM dues_plans WHERE id = $1`

	p := &models.DuesPlan{}
	err := r.pool.QueryRow(ctx, query, planID).Scan(
		&p.ID, &p.BuildingID, &p.Title, &p.Amount,
		&p.PeriodMonth, &p.PeriodYear, &p.DueDate, &p.CreatedBy, &p.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("dues plan not found")
		}
		return nil, err
	}
	return p, nil
}

// Payments

func (r *FinancialRepository) CreatePayment(ctx context.Context, payment *models.DuePayment) error {
	now := time.Now()
	payment.PaidAt = &now
	payment.Status = models.PaymentPaid

	query := `
		INSERT INTO due_payments (dues_plan_id, unit_id, paid_amount, paid_at, status, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (dues_plan_id, unit_id) DO UPDATE SET
			paid_amount = $3, paid_at = $4, status = $5, notes = $6
		RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		payment.DuesPlanID, payment.UnitID, payment.PaidAmount, payment.PaidAt, payment.Status, payment.Notes,
	).Scan(&payment.ID, &payment.CreatedAt, &payment.UpdatedAt)
}

func (r *FinancialRepository) GetDuesReport(ctx context.Context, buildingID uuid.UUID, month, year int) (*models.DuesReport, error) {
	report := &models.DuesReport{
		BuildingID:  buildingID,
		PeriodMonth: month,
		PeriodYear:  year,
	}

	query := `
		SELECT dp.id, dp.dues_plan_id, dp.unit_id, dp.paid_amount, dp.paid_at, dp.receipt_url,
			dp.status, dp.notes, dp.created_at, dp.updated_at,
			u.unit_number, COALESCE(usr.full_name, '') as owner_name
		FROM due_payments dp
		JOIN dues_plans p ON dp.dues_plan_id = p.id
		JOIN units u ON dp.unit_id = u.id
		LEFT JOIN users usr ON u.owner_id = usr.id
		WHERE p.building_id = $1 AND p.period_month = $2 AND p.period_year = $3
		ORDER BY u.unit_number`

	rows, err := r.pool.Query(ctx, query, buildingID, month, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var d models.DuePaymentDetail
		if err := rows.Scan(
			&d.ID, &d.DuesPlanID, &d.UnitID, &d.PaidAmount, &d.PaidAt,
			&d.ReceiptURL, &d.Status, &d.Notes, &d.CreatedAt, &d.UpdatedAt,
			&d.UnitNumber, &d.OwnerName,
		); err != nil {
			return nil, err
		}

		switch d.Status {
		case models.PaymentPaid:
			report.PaidCount++
			if d.PaidAmount != nil {
				report.TotalPaid += *d.PaidAmount
			}
		case models.PaymentPending:
			report.PendingCount++
		case models.PaymentLate:
			report.LateCount++
		}

		report.Payments = append(report.Payments, d)
	}

	// Get total due amount
	err = r.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM dues_plans WHERE building_id = $1 AND period_month = $2 AND period_year = $3`,
		buildingID, month, year,
	).Scan(&report.TotalDue)
	if err != nil {
		return nil, err
	}

	report.TotalPending = report.TotalDue - report.TotalPaid

	return report, nil
}

func (r *FinancialRepository) UpdateDuesPlan(ctx context.Context, planID uuid.UUID, req *models.UpdateDuesPlanRequest) (*models.DuesPlan, error) {
	var dueDate *time.Time
	if req.DueDate != nil {
		parsed, err := time.Parse("2006-01-02", *req.DueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date format (YYYY-MM-DD)")
		}
		dueDate = &parsed
	}

	query := `
		UPDATE dues_plans SET
			title = COALESCE($2, title),
			amount = COALESCE($3, amount),
			period_month = COALESCE($4, period_month),
			period_year = COALESCE($5, period_year),
			due_date = COALESCE($6, due_date)
		WHERE id = $1
		RETURNING id, building_id, title, amount, period_month, period_year, due_date, created_by, created_at`

	p := &models.DuesPlan{}
	err := r.pool.QueryRow(ctx, query, planID, req.Title, req.Amount, req.PeriodMonth, req.PeriodYear, dueDate).Scan(
		&p.ID, &p.BuildingID, &p.Title, &p.Amount,
		&p.PeriodMonth, &p.PeriodYear, &p.DueDate, &p.CreatedBy, &p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *FinancialRepository) DeleteDuesPlan(ctx context.Context, planID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM dues_plans WHERE id = $1`, planID)
	return err
}

func (r *FinancialRepository) UpdateExpense(ctx context.Context, expenseID uuid.UUID, req *models.UpdateExpenseRequest) (*models.Expense, error) {
	var date *time.Time
	if req.Date != nil {
		parsed, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date format (YYYY-MM-DD)")
		}
		date = &parsed
	}

	query := `
		UPDATE expenses SET
			category = COALESCE($2, category),
			amount = COALESCE($3, amount),
			description = COALESCE($4, description),
			date = COALESCE($5, date)
		WHERE id = $1
		RETURNING id, building_id, category, amount, description, date, receipt_url, created_by, created_at`

	e := &models.Expense{}
	err := r.pool.QueryRow(ctx, query, expenseID, req.Category, req.Amount, req.Description, date).Scan(
		&e.ID, &e.BuildingID, &e.Category, &e.Amount,
		&e.Description, &e.Date, &e.ReceiptURL, &e.CreatedBy, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *FinancialRepository) DeleteExpense(ctx context.Context, expenseID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM expenses WHERE id = $1`, expenseID)
	return err
}

// Expenses

func (r *FinancialRepository) CreateExpense(ctx context.Context, expense *models.Expense) error {
	query := `
		INSERT INTO expenses (building_id, category, amount, description, date, receipt_url, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`

	return r.pool.QueryRow(ctx, query,
		expense.BuildingID, expense.Category, expense.Amount, expense.Description,
		expense.Date, expense.ReceiptURL, expense.CreatedBy,
	).Scan(&expense.ID, &expense.CreatedAt)
}

func (r *FinancialRepository) GetExpenses(ctx context.Context, buildingID uuid.UUID, page, limit int) ([]models.Expense, int64, error) {
	var total int64
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM expenses WHERE building_id = $1`, buildingID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, building_id, category, amount, description, date, receipt_url, created_by, created_at
		FROM expenses WHERE building_id = $1
		ORDER BY date DESC
		LIMIT $2 OFFSET $3`

	offset := (page - 1) * limit
	rows, err := r.pool.Query(ctx, query, buildingID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var expenses []models.Expense
	for rows.Next() {
		var e models.Expense
		if err := rows.Scan(
			&e.ID, &e.BuildingID, &e.Category, &e.Amount,
			&e.Description, &e.Date, &e.ReceiptURL, &e.CreatedBy, &e.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		expenses = append(expenses, e)
	}
	return expenses, total, nil
}
