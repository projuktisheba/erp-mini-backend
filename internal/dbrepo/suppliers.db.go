package dbrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

type SupplierRepo struct {
	db *pgxpool.Pool
}

func NewSupplierRepo(db *pgxpool.Pool) *SupplierRepo {
	return &SupplierRepo{db: db}
}

// CreateSupplier inserts a new supplier
func (r *SupplierRepo) CreateSupplier(ctx context.Context, s *models.Supplier) error {
	query := `
		INSERT INTO suppliers (name, branch_id, status, mobile, created_at, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`

	row := r.db.QueryRow(ctx, query, s.Name, s.BranchID, s.Status, s.Mobile)

	err := row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			if pgErr.ConstraintName == "suppliers_mobile_key" {
				return errors.New("this mobile is already associated with another supplier")
			}
		}
		if err == pgx.ErrNoRows {
			return errors.New("failed to insert supplier")
		}
		return err
	}

	return nil
}

// UpdateSupplier updates supplier details
func (r *SupplierRepo) UpdateSupplier(ctx context.Context, s *models.Supplier) error {
	query := `
		UPDATE suppliers
		SET name = $2,
		    status = $3,
		    mobile = $4,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at
	`

	row := r.db.QueryRow(ctx, query, s.ID, s.Name, s.Status, s.Mobile)

	err := row.Scan(&s.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "suppliers_mobile_key" {
				return errors.New("this mobile is already associated with another supplier")
			}
		}
		if err == pgx.ErrNoRows {
			return errors.New("no supplier found with the given id")
		}
		return err
	}

	return nil
}

// GetSupplierByID fetches a supplier by its ID
func (r *SupplierRepo) GetSupplierByID(ctx context.Context, id int64) (*models.Supplier, error) {
	query := `
		SELECT id, name, branch_id, status, mobile, created_at, updated_at
		FROM suppliers
		WHERE id = $1
	`

	row := r.db.QueryRow(ctx, query, id)

	s := &models.Supplier{}
	err := row.Scan(&s.ID, &s.Name, &s.BranchID, &s.Status, &s.Mobile, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no supplier found
		}
		return nil, err
	}

	return s, nil
}

func (r *SupplierRepo) ListSuppliers(ctx context.Context, name, status, mobile string, page, limit int, branchID int64) ([]*models.Supplier, int, error) {
	baseQuery := `
		SELECT id, name, branch_id, status, mobile, created_at, updated_at
		FROM suppliers
	`
	conditions := []string{}
	args := []interface{}{}
	argPos := 1

	if name != "" {
		conditions = append(conditions, fmt.Sprintf("name ILIKE '%%' || $%d || '%%'", argPos))
		args = append(args, name)
		argPos++
	}
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argPos))
		args = append(args, status)
		argPos++
	}
	if mobile != "" {
		conditions = append(conditions, fmt.Sprintf("mobile = $%d", argPos))
		args = append(args, mobile)
		argPos++
	}
	if branchID > 0 {
		conditions = append(conditions, fmt.Sprintf("branch_id = $%d", argPos))
		args = append(args, branchID)
		argPos++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total rows
	countQuery := "SELECT COUNT(*) FROM suppliers" + whereClause
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := baseQuery + whereClause + " ORDER BY name"

	// Add pagination only if limit > 0
	if limit > 0 {
		offset := (page - 1) * limit
		args = append(args, limit, offset)
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argPos, argPos+1)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var suppliers []*models.Supplier
	for rows.Next() {
		s := &models.Supplier{}
		if err := rows.Scan(&s.ID, &s.Name, &s.BranchID, &s.Status, &s.Mobile, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		suppliers = append(suppliers, s)
	}

	return suppliers, total, nil
}

