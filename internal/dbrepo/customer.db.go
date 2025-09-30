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
// It returns a custom error if the mobile or tax_id is already in use.
func (s *CustomerRepo) CreateNewCustomer(ctx context.Context, customer *models.Customer) error {
	query := `
		INSERT INTO customers (name, mobile, address, tax_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, status, due_amount, created_at, updated_at;`

	// Prepare arguments, handling potentially null strings
	args := []interface{}{
		customer.Name,
		customer.Mobile,
		customer.Address,
		customer.TaxID,
	}

	// The context is used for cancellation or timeouts.
	err := s.db.QueryRow(ctx, query, args...).Scan(
		&customer.ID,
		&customer.Status,
		&customer.DueAmount,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError
		// Use errors.As to check if the error is a pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // 23505 is the code for unique_violation
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

// 2. UpdateCustomerInfo updates a customer's basic information (name, mobile, address, tax_id).
// It does not update status or due amount.
func (s *CustomerRepo) UpdateCustomerInfo(ctx context.Context, customer *models.Customer) (*time.Time, error) {
	query := `
		UPDATE customers
		SET name = $1, mobile = $2, address = $3, tax_id = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING updated_at;`

	var updatedAt time.Time
	err := s.db.QueryRow(ctx, query, customer.Name, customer.Mobile, customer.Address, customer.TaxID, customer.ID).Scan(&updatedAt)
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

// 3. UpdateCustomerDueAmount updates only the due_amount for a specific customer.
func (s *CustomerRepo) UpdateCustomerDueAmount(ctx context.Context, id int64, dueAmount float64) error {
	query := `UPDATE customers SET due_amount = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2;`

	res, err := s.db.Exec(ctx, query, dueAmount, id)
	if err != nil {
		return fmt.Errorf("error updating due amount: %w", err)
	}

	rowsAffected := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("customer with id %d not found", id)
	}

	return nil
}

// 4. UpdateCustomerStatus updates the active/inactive status of a customer.
func (s *CustomerRepo) UpdateCustomerStatus(ctx context.Context, id int64, status bool) error {
	query := `UPDATE customers SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2;`

	res, err := s.db.Exec(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("error updating status: %w", err)
	}

	rowsAffected := res.RowsAffected()

	if rowsAffected == 0 {
		return fmt.Errorf("customer with id %d not found", id)
	}

	return nil
}

// 5. GetCustomerByID fetches a single customer by their primary key (ID).
func (s *CustomerRepo) GetCustomerByID(ctx context.Context, id int64) (*models.Customer, error) {
	return s.getCustomerBy(ctx, "id", id)
}

// GetCustomerByMobile fetches a single customer by their unique mobile number.
func (s *CustomerRepo) GetCustomerByMobile(ctx context.Context, mobile string) (*models.Customer, error) {
	return s.getCustomerBy(ctx, "mobile", mobile)
}

// GetCustomerByTaxID fetches a single customer by their unique tax ID.
func (s *CustomerRepo) GetCustomerByTaxID(ctx context.Context, taxID string) (*models.Customer, error) {
	return s.getCustomerBy(ctx, "tax_id", taxID)
}

// getCustomerBy is a helper function to fetch a single customer by a specific unique field.
func (s *CustomerRepo) getCustomerBy(ctx context.Context, field string, value any) (*models.Customer, error) {
	query := fmt.Sprintf(`
		SELECT id, name, mobile, address, tax_id, due_amount, status, created_at, updated_at
		FROM customers
		WHERE %s = $1;`, field)

	c := &models.Customer{}
	err := s.db.QueryRow(ctx, query, value).Scan(
		&c.ID, &c.Name, &c.Mobile, &c.Address, &c.TaxID, &c.DueAmount, &c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, fmt.Errorf("error fetching customer by %s: %w", field, err)
	}
	return c, nil
}

// 6. FilterCustomersByName searches for customers using a case-insensitive name match.
// It returns a slice of customers.
func (s *CustomerRepo) FilterCustomersByName(ctx context.Context, name string) ([]*models.Customer, error) {
	query := `
		SELECT id, name, mobile, address, tax_id, due_amount, status, created_at, updated_at
		FROM customers
		WHERE name ILIKE $1
		ORDER BY name ASC;`

	// Use '%' for partial matching
	rows, err := s.db.Query(ctx, query, "%"+name+"%")
	if err != nil {
		return nil, fmt.Errorf("error filtering customers by name: %w", err)
	}
	defer rows.Close()

	customers := []*models.Customer{}
	for rows.Next() {
		c := &models.Customer{}
		if err := rows.Scan(&c.ID, &c.Name, &c.Mobile, &c.Address, &c.TaxID, &c.DueAmount, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning customer row: %w", err)
		}
		customers = append(customers, c)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer rows: %w", err)
	}

	return customers, nil
}

// GetCustomersFilter defines the parameters for filtering customers.
// 7. GetCustomers fetches a list of customers with filtering and pagination.
// If limit is -1, it returns all customers, ignoring page and other filters.
func (s *CustomerRepo) GetCustomers(ctx context.Context, page, limit int) ([]*models.Customer, error) {
	if limit == -1 {
		// fetch all
		query := `
			SELECT id, name, mobile, address, tax_id, due_amount, status, created_at, updated_at
			FROM customers
			ORDER BY created_at DESC;`

		rows, err := s.db.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("error fetching customers: %w", err)
		}
		defer rows.Close()

		var customers []*models.Customer
		for rows.Next() {
			var c models.Customer
			if err := rows.Scan(&c.ID, &c.Name, &c.Mobile, &c.Address, &c.TaxID, &c.DueAmount, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
				return nil, fmt.Errorf("error scanning customer: %w", err)
			}
			customers = append(customers, &c)
		}
		return customers, nil
	}

	// paginated
	offset := (page - 1) * limit
	query := `
		SELECT id, name, mobile, address, tax_id, due_amount, status, created_at, updated_at
		FROM customers
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2;`

	rows, err := s.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("error fetching customers: %w", err)
	}
	defer rows.Close()

	var customers []*models.Customer
	for rows.Next() {
		var c models.Customer
		if err := rows.Scan(&c.ID, &c.Name, &c.Mobile, &c.Address, &c.TaxID, &c.DueAmount, &c.Status, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("error scanning customer: %w", err)
		}
		customers = append(customers, &c)
	}

	return customers, nil
}

// 8. GetCustomersNameAndID fetches a lightweight list of all active customers (ID and Name only).
func (s *CustomerRepo) GetCustomersNameAndID(ctx context.Context) ([]*models.CustomerNameID, error) {
	query := `SELECT id, name FROM customers WHERE status = TRUE ORDER BY name ASC;`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("error getting customer names and ids: %w", err)
	}
	defer rows.Close()

	list := []*models.CustomerNameID{}
	for rows.Next() {
		var item models.CustomerNameID
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("error scanning customer name/id: %w", err)
		}
		list = append(list, &item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer name/id rows: %w", err)
	}

	return list, nil
}
func (r *CustomerRepo) GetCustomersWithDue(ctx context.Context) ([]*models.Customer, error) {
	query := `
		SELECT id, name, mobile, address, tax_id, due_amount, status, created_at, updated_at
		FROM customers
		WHERE due_amount > 0
		ORDER BY due_amount DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []*models.Customer
	for rows.Next() {
		var c models.Customer
		if err := rows.Scan(
			&c.ID, &c.Name, &c.Mobile, &c.Address, &c.TaxID,
			&c.DueAmount, &c.Status, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		customers = append(customers, &c)
	}

	return customers, nil
}
