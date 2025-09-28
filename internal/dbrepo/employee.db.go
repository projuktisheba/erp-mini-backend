package dbrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

// ============================== User Repository ==============================
type EmployeeRepo struct {
	db *pgxpool.Pool
}

func NewEmployeeRepo(db *pgxpool.Pool) *EmployeeRepo {
	return &EmployeeRepo{db: db}
}

// CreateEmployee inserts a new employee
func (user *EmployeeRepo) CreateEmployee(ctx context.Context, e *models.Employee) error {
	query := `
		INSERT INTO employees 
		(fname, lname, role, status, bio, email, password, mobile, country, city, postal_code, tax_id, base_salary, overtime_rate, avatar_link, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at;
	`
	return user.db.QueryRow(ctx, query,
		e.FirstName, e.LastName, e.Role, e.Status, e.Bio, e.Email, e.Password,
		e.Mobile, e.Country, e.City, e.PostalCode, e.TaxID,
		e.BaseSalary, e.OvertimeRate, e.AvatarLink,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
}

// GetEmployee fetches an employee by ID
func (user *EmployeeRepo) GetEmployee(ctx context.Context, id int) (*models.Employee, error) {
	query := `SELECT id, fname, lname, role, status, bio, email, mobile, country, city, postal_code, tax_id, base_salary, overtime_rate, avatar_link, created_at, updated_at FROM employees WHERE id=$1`
	e := &models.Employee{}
	err := user.db.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.FirstName, &e.LastName, &e.Role, &e.Status, &e.Bio, &e.Email,
		&e.Mobile, &e.Country, &e.City, &e.PostalCode, &e.TaxID,
		&e.BaseSalary, &e.OvertimeRate, &e.AvatarLink,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("No user found")
		}
		return nil, err
	}
	return e, nil
}

// GetEmployee fetches an employee by email
func (user *EmployeeRepo) GetEmployeeByEmail(ctx context.Context, email string) (*models.Employee, error) {
	query := fmt.Sprintf(`SELECT id, fname, lname, role, status, bio, email, password, mobile, country, city, postal_code, tax_id, base_salary, overtime_rate, avatar_link, created_at, updated_at FROM employees WHERE email='%s'`, email)
	e := &models.Employee{}
	err := user.db.QueryRow(ctx, query).Scan(
		&e.ID, &e.FirstName, &e.LastName, &e.Role, &e.Status, &e.Bio, &e.Email, &e.Password,
		&e.Mobile, &e.Country, &e.City, &e.PostalCode, &e.TaxID,
		&e.BaseSalary, &e.OvertimeRate, &e.AvatarLink,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("No user found")
		}
		return nil, err
	}
	return e, nil
}

// UpdateEmployee updates employee details
func (user *EmployeeRepo) UpdateEmployee(ctx context.Context, e *models.Employee) error {
	query := `
		UPDATE employees
		SET fname=$1, lname=$2, bio=$3, mobile=$4, country=$5, city=$6, postal_code=$7, tax_id=$8,  avatar_link=$9, updated_at= CURRENT_TIMESTAMP
		WHERE id=$10
		RETURNING updated_at;
	`
	return user.db.QueryRow(ctx, query,
		e.FirstName, e.LastName, e.Bio, e.Mobile,
		e.Country, e.City, e.PostalCode, e.TaxID, e.AvatarLink, e.ID,
	).Scan(&e.UpdatedAt)
}

// UpdateEmployeeSalary updates employee salary and overtime rate
// Call this function if the role of the token user is Admin
func (user *EmployeeRepo) UpdateEmployeeSalary(ctx context.Context, e *models.Employee) error {
	query := `
		UPDATE employees
		SET base_salary=$1, overtime_rate=$2, updated_at= CURRENT_TIMESTAMP
		WHERE id=$3
		RETURNING updated_at;
	`
	return user.db.QueryRow(ctx, query,
		e.BaseSalary,
		e.OvertimeRate,
		e.ID,
	).Scan(&e.UpdatedAt)
}

// UpdateEmployeeStatus updates employee role and status
// Call this function if the role of the token user is Admin
func (user *EmployeeRepo) UpdateEmployeeRole(ctx context.Context, e *models.Employee) error {
	query := `
		UPDATE employees
		SET role =$1, status=$2, updated_at= CURRENT_TIMESTAMP
		WHERE id=$3
		RETURNING updated_at;
	`
	return user.db.QueryRow(ctx, query,
		e.Role,
		e.Status,
		e.ID,
	).Scan(&e.UpdatedAt)
}

// DeleteEmployee removes an employee by ID
func (user *EmployeeRepo) DeleteEmployee(ctx context.Context, id int) error {
	_, err := user.db.Exec(ctx, "DELETE FROM employees WHERE id=$1", id)
	return err
}

// ListEmployees fetches all employees
func (user *EmployeeRepo) ListEmployees(ctx context.Context) ([]*models.Employee, error) {
	query := `SELECT id, fname, lname, role, status, bio, email, mobile, country, city, postal_code, tax_id, base_salary, overtime_rate, avatar_link, created_at, updated_at FROM employees ORDER BY id`
	rows, err := user.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var employees []*models.Employee
	for rows.Next() {
		var e models.Employee
		err := rows.Scan(
			&e.ID, &e.FirstName, &e.LastName, &e.Role, &e.Status, &e.Bio, &e.Email,
			&e.Mobile, &e.Country, &e.City, &e.PostalCode, &e.TaxID,
			&e.BaseSalary, &e.OvertimeRate, &e.AvatarLink,
			&e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		employees = append(employees, &e)
	}
	return employees, nil
}

// PaginatedEmployeeList returns a paginated list of employees with optional role & status filters.
// PaginatedEmployeeList returns a paginated list of employees with optional role & status filters.
func (e *EmployeeRepo) PaginatedEmployeeList(ctx context.Context, page, limit int, role, status string) ([]*models.Employee, int, error) {
	offset := (page - 1) * limit

	// Base queries
	query := `SELECT id, fname, lname, role, status, bio, email, mobile,
	                 country, city, postal_code, tax_id, base_salary, overtime_rate,
	                 avatar_link, created_at, updated_at
	          FROM employees
	          WHERE 1=1`

	countQuery := `SELECT COUNT(*) FROM employees WHERE 1=1`

	// Dynamic filters
	args := []interface{}{}
	countArgs := []interface{}{}
	argIdx := 1

	if role != "" {
		query += fmt.Sprintf(" AND role = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND role = $%d", argIdx)
		args = append(args, role)
		countArgs = append(countArgs, role)
		argIdx++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		countArgs = append(countArgs, status)
		argIdx++
	}

	// Add pagination
	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	// Get total count (without limit/offset)
	var total int
	if err := e.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Query employees
	rows, err := e.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	employees := []*models.Employee{}
	for rows.Next() {
		var emp models.Employee
		err := rows.Scan(
			&emp.ID, &emp.FirstName, &emp.LastName, &emp.Role, &emp.Status,
			&emp.Bio, &emp.Email, &emp.Mobile, &emp.Country, &emp.City,
			&emp.PostalCode, &emp.TaxID, &emp.BaseSalary, &emp.OvertimeRate,
			&emp.AvatarLink, &emp.CreatedAt, &emp.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		employees = append(employees, &emp)
	}

	return employees, total, nil
}
