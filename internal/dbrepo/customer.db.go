package dbrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

// ============================== Customer Repository ==============================
type CustomerRepo struct {
	db *pgxpool.Pool
}

func NewCustomerRepo(db *pgxpool.Pool) *CustomerRepo {
	return &CustomerRepo{db: db}
}

// 1. CreateNewCustomer adds a new customer to the database.
func (s *CustomerRepo) CreateNewCustomer(ctx context.Context, customer *models.Customer) error {
	query := `
		INSERT INTO customers 
		(name, mobile, address, tax_id, branch_id,
		 length, shoulder, bust, waist, hip,
		 arm_hole, sleeve_length, sleeve_width, round_width)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, status, due_amount, created_at, updated_at;`

	args := []interface{}{
		customer.Name,
		customer.Mobile,
		customer.Address,
		customer.TaxID,
		customer.BranchID,
		customer.Length,
		customer.Shoulder,
		customer.Bust,
		customer.Waist,
		customer.Hip,
		customer.ArmHole,
		customer.SleeveLength,
		customer.SleeveWidth,
		customer.RoundWidth,
	}

	err := s.db.QueryRow(ctx, query, args...).Scan(
		&customer.ID,
		&customer.Status,
		&customer.DueAmount,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "customers_mobile_key":
				return errors.New("this mobile is already associated with another account")
			case "customers_tax_id_key":
				return errors.New("this tax id is already associated with another account")
			}
		}
		return fmt.Errorf("error creating customer: %w", err)
	}
	return nil
}

// 2. UpdateCustomerInfo updates a customer's basic information.
func (s *CustomerRepo) UpdateCustomerInfo(ctx context.Context, customer *models.Customer) (*time.Time, error) {
	query := `
		UPDATE customers
		SET name = $1, mobile = $2, address = $3, tax_id = $4,
		    length = $5, shoulder = $6, bust = $7, waist = $8, hip = $9,
		    arm_hole = $10, sleeve_length = $11, sleeve_width = $12, round_width = $13,
		    updated_at = NOW()
		WHERE id = $14
		RETURNING updated_at;`

	var updatedAt time.Time
	err := s.db.QueryRow(ctx, query,
		customer.Name, customer.Mobile, customer.Address, customer.TaxID,
		customer.Length, customer.Shoulder, customer.Bust, customer.Waist, customer.Hip,
		customer.ArmHole, customer.SleeveLength, customer.SleeveWidth, customer.RoundWidth,
		customer.ID,
	).Scan(&updatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("customer with id %d not found", customer.ID)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "customers_mobile_key":
				return nil, errors.New("this mobile is already associated with another account")
			case "customers_tax_id_key":
				return nil, errors.New("this tax id is already associated with another account")
			}
		}
		return nil, fmt.Errorf("error updating customer info: %w", err)
	}

	return &updatedAt, nil
}

// 3. DeductCustomerDueAmount deducts the due amount of a customer.
func (s *CustomerRepo) DeductCustomerDueAmount(ctx context.Context, customerID, branchID int64, deductedAmount float64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	query := `UPDATE customers SET due_amount = due_amount - $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2;`

	res, err := tx.Exec(ctx, query, deductedAmount, customerID)
	if err != nil {
		return fmt.Errorf("error updating due amount: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("customer with id %d not found", customerID)
	}

	err = SaveTopSheetTx(tx, ctx, &models.TopSheet{
		Date:     time.Now(),
		BranchID: branchID,
		Cash:     deductedAmount,
	})
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// 4. UpdateCustomerStatus updates active/inactive status.
func (s *CustomerRepo) UpdateCustomerStatus(ctx context.Context, id int64, status bool) error {
	query := `UPDATE customers SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2;`

	res, err := s.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("error updating status: %w", err)
	}

	if res.RowsAffected() == 0 {
		return fmt.Errorf("customer with id %d not found", id)
	}
	return nil
}

// 5. GetCustomerByID
func (s *CustomerRepo) GetCustomerByID(ctx context.Context, id int64) (*models.Customer, error) {
	return s.getCustomerBy(ctx, "id", id)
}

// GetCustomerByMobile
func (s *CustomerRepo) GetCustomerByMobile(ctx context.Context, mobile string) (*models.Customer, error) {
	return s.getCustomerBy(ctx, "mobile", mobile)
}

// GetCustomerByTaxID
func (s *CustomerRepo) GetCustomerByTaxID(ctx context.Context, taxID string) (*models.Customer, error) {
	return s.getCustomerBy(ctx, "tax_id", taxID)
}

// getCustomerBy helper
func (s *CustomerRepo) getCustomerBy(ctx context.Context, field string, value any) (*models.Customer, error) {
	query := fmt.Sprintf(`
		SELECT id, name, mobile, address, tax_id, branch_id, due_amount, status,
		       length, shoulder, bust, waist, hip, arm_hole,
		       sleeve_length, sleeve_width, round_width,
		       created_at, updated_at
		FROM customers
		WHERE %s = $1;`, field)

	c := &models.Customer{}
	err := s.db.QueryRow(ctx, query, value).Scan(
		&c.ID, &c.Name, &c.Mobile, &c.Address, &c.TaxID, &c.BranchID, &c.DueAmount, &c.Status,
		&c.Length, &c.Shoulder, &c.Bust, &c.Waist, &c.Hip, &c.ArmHole,
		&c.SleeveLength, &c.SleeveWidth, &c.RoundWidth,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error fetching customer by %s: %w", field, err)
	}
	return c, nil
}

// 6. FilterCustomersByName (ILIKE search)
func (s *CustomerRepo) FilterCustomersByName(ctx context.Context, branchID int64, name string) ([]*models.Customer, error) {
	query := `
		SELECT id, name, mobile, address, tax_id, branch_id, due_amount, status,
		       length, shoulder, bust, waist, hip, arm_hole,
		       sleeve_length, sleeve_width, round_width,
		       created_at, updated_at
		FROM customers
		WHERE branch_id = $1 AND name ILIKE $2
		ORDER BY name ASC;`

	rows, err := s.db.Query(ctx, query, branchID, "%"+name+"%")
	if err != nil {
		return nil, fmt.Errorf("error filtering customers by name: %w", err)
	}
	defer rows.Close()

	var customers []*models.Customer
	for rows.Next() {
		var c models.Customer
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Mobile, &c.Address, &c.TaxID, &c.BranchID, &c.DueAmount, &c.Status,
			&c.Length, &c.Shoulder, &c.Bust, &c.Waist, &c.Hip, &c.ArmHole,
			&c.SleeveLength, &c.SleeveWidth, &c.RoundWidth,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning customer row: %w", err)
		}
		customers = append(customers, &c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer rows: %w", err)
	}

	// Return explicit no rows when empty for callers that want to distinguish
	if len(customers) == 0 {
		return nil, pgx.ErrNoRows
	}
	return customers, nil
}

// 7. GetCustomers (with pagination or all)
func (s *CustomerRepo) GetCustomers(ctx context.Context, page, limit int, branchID int64) ([]*models.Customer, error) {
	var query string
	var rows pgx.Rows
	var err error

	if limit == -1 {
		query = `
			SELECT id, name, mobile, address, tax_id, branch_id, due_amount, status,
			       length, shoulder, bust, waist, hip, arm_hole,
			       sleeve_length, sleeve_width, round_width,
			       created_at, updated_at
			FROM customers
			WHERE branch_id = $1
			ORDER BY created_at DESC;`
		rows, err = s.db.Query(ctx, query, branchID)
	} else {
		offset := (page - 1) * limit
		query = `
			SELECT id, name, mobile, address, tax_id, branch_id, due_amount, status,
			       length, shoulder, bust, waist, hip, arm_hole,
			       sleeve_length, sleeve_width, round_width,
			       created_at, updated_at
			FROM customers
			WHERE branch_id = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3;`
		rows, err = s.db.Query(ctx, query, branchID, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("error fetching customers: %w", err)
	}
	defer rows.Close()

	var customers []*models.Customer
	for rows.Next() {
		var c models.Customer
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Mobile, &c.Address, &c.TaxID, &c.BranchID, &c.DueAmount, &c.Status,
			&c.Length, &c.Shoulder, &c.Bust, &c.Waist, &c.Hip, &c.ArmHole,
			&c.SleeveLength, &c.SleeveWidth, &c.RoundWidth,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("error scanning customer: %w", err)
		}
		customers = append(customers, &c)
	}
	if len(customers) == 0 {
		return nil, pgx.ErrNoRows
	}
	return customers, nil
}

// 8. GetCustomersNameAndID (active only)
func (s *CustomerRepo) GetCustomersNameAndID(ctx context.Context, branchID int64) ([]*models.CustomerNameID, error) {
	query := `SELECT id, name, mobile FROM customers WHERE status = TRUE AND branch_id=$1 ORDER BY name ASC;`

	rows, err := s.db.Query(ctx, query, branchID)
	if err != nil {
		return nil, fmt.Errorf("error getting customer names and ids: %w", err)
	}
	defer rows.Close()

	var list []*models.CustomerNameID
	for rows.Next() {
		var item models.CustomerNameID
		if err := rows.Scan(&item.ID, &item.Name, &item.Mobile); err != nil {
			return nil, fmt.Errorf("error scanning customer name/id: %w", err)
		}
		list = append(list, &item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer name/id rows: %w", err)
	}
	return list, nil
}

// 9. GetCustomersWithDue
func (s *CustomerRepo) GetCustomersWithDue(ctx context.Context, branchID int64) ([]*models.Customer, error) {
	query := `
		SELECT id, name, mobile, address, tax_id, branch_id, due_amount, status,
		       length, shoulder, bust, waist, hip, arm_hole,
		       sleeve_length, sleeve_width, round_width,
		       created_at, updated_at
		FROM customers
		WHERE due_amount > 0 AND branch_id=$1
		ORDER BY due_amount DESC;`

	rows, err := s.db.Query(ctx, query, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []*models.Customer
	for rows.Next() {
		var c models.Customer
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Mobile, &c.Address, &c.TaxID, &c.BranchID, &c.DueAmount, &c.Status,
			&c.Length, &c.Shoulder, &c.Bust, &c.Waist, &c.Hip, &c.ArmHole,
			&c.SleeveLength, &c.SleeveWidth, &c.RoundWidth,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		customers = append(customers, &c)
	}
	return customers, nil
}
