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

// ============================== Employee Repository ==============================
type EmployeeRepo struct {
	db *pgxpool.Pool
}

func NewEmployeeRepo(db *pgxpool.Pool) *EmployeeRepo {
	return &EmployeeRepo{db: db}
}

func (r *EmployeeRepo) CreateEmployee(ctx context.Context, e *models.Employee) error {
	query := `
		INSERT INTO employees 
		(name, role, mobile, email, password, passport_no, joining_date, address, base_salary, overtime_rate, branch_id, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`

	row := r.db.QueryRow(ctx, query,
		e.Name, e.Role, e.Mobile, e.Email, e.Password, e.PassportNo,
		e.JoiningDate, e.Address, e.BaseSalary, e.OvertimeRate, e.BranchID,
	)

	err := row.Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			switch pgErr.ConstraintName {
			case "employees_mobile_key":
				return errors.New("this mobile is already associated with another account")
			case "employees_email_key":
				return errors.New("this email is already associated with another account")
			}
		}
		if err == pgx.ErrNoRows {
			return errors.New("failed to insert employee")
		}
		return err
	}

	return nil
}

// GetEmployee fetches an employee by ID
func (user *EmployeeRepo) GetEmployeeByID(ctx context.Context, id int64) (*models.Employee, error) {
	query := `
		SELECT 
			id, name, role, mobile, email, password, passport_no, joining_date, address, 
			base_salary, overtime_rate, branch_id, created_at, updated_at
		FROM employees 
		WHERE id = $1
	`
	e := &models.Employee{}
	err := user.db.QueryRow(ctx, query, id).Scan(
		&e.ID, &e.Name, &e.Role, &e.Mobile, &e.Email, &e.Password,
		&e.PassportNo, &e.JoiningDate, &e.Address,
		&e.BaseSalary, &e.OvertimeRate, &e.BranchID,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("No employee found")
		}
		return nil, err
	}
	return e, nil
}

// GetEmployeeByUsernameOrMobile fetches an employee by mobile or email
func (user *EmployeeRepo) GetEmployeeByUsernameOrMobile(ctx context.Context, username string) (*models.Employee, error) {
	query := `
		SELECT 
			id, name, role, mobile, email, password, passport_no, joining_date, address, 
			base_salary, overtime_rate, branch_id, created_at, updated_at
		FROM employees 
		WHERE mobile = $1 OR email = $1
		LIMIT 1
	`
	e := &models.Employee{}
	err := user.db.QueryRow(ctx, query, username).Scan(
		&e.ID, &e.Name, &e.Role, &e.Mobile, &e.Email, &e.Password,
		&e.PassportNo, &e.JoiningDate, &e.Address,
		&e.BaseSalary, &e.OvertimeRate, &e.BranchID,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, errors.New("No employee found")
		}
		return nil, err
	}
	return e, nil
}

// UpdateEmployee updates employee details
func (r *EmployeeRepo) UpdateEmployee(ctx context.Context, e *models.Employee) error {
	query := `
		UPDATE employees
		SET 
			name = $2,
			mobile = $3,
			email = $4,
			password = $5,
			passport_no = $6,
			joining_date = $7,
			address = $8,
			base_salary = $9,
			overtime_rate = $10,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		RETURNING updated_at
	`

	row := r.db.QueryRow(ctx, query,
		e.ID, e.Name, e.Mobile, e.Email, e.Password, e.PassportNo,
		e.JoiningDate, e.Address, e.BaseSalary, e.OvertimeRate,
	)

	err := row.Scan(&e.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" { // unique_violation
				switch pgErr.ConstraintName {
				case "employees_mobile_key":
					return errors.New("this mobile is already associated with another employee")
				case "employees_email_key":
					return errors.New("this email is already associated with another employee")
				default:
					return errors.New("unique constraint violation: " + pgErr.Message)
				}
			}
			// fallback for other Postgres-specific errors
			return errors.New("database error: " + pgErr.Message)
		}

		if err == pgx.ErrNoRows {
			return errors.New("no employee found with the given id")
		}

		return err
	}

	return nil
}

// UpdateEmployeeAvatarLink updates employee avatar_link field
func (user *EmployeeRepo) UpdateEmployeeAvatarLink(ctx context.Context, id int, avatarLink string) error {
	query := `
		UPDATE employees
		SET avatar_link=$1, updated_at= CURRENT_TIMESTAMP
		WHERE id=$2
		RETURNING updated_at;
	`
	_, err := user.db.Exec(ctx, query, avatarLink, id)
	return err
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

// SubmitSalary generates and give employee salary
// Call this function if the role of the token user is Admin
func (user *EmployeeRepo) SubmitSalary(ctx context.Context, salaryDate time.Time, employeeID, branchID int64, amount float64) error {
	//using pgxpool begin a transaction
	tx, err := user.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) // will rollback if not committed

	// Insert or update in attendance table
	query := `
		INSERT INTO attendance (
			employee_id, work_date, branch_id, status, advance_payment, overtime_hours, production_units
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (employee_id, work_date)
		DO UPDATE SET 
			status          = EXCLUDED.status,  -- replace status
			advance_payment = attendance.advance_payment + EXCLUDED.advance_payment,
			overtime_hours  = attendance.overtime_hours + EXCLUDED.overtime_hours,
			production_units = attendance.production_units + EXCLUDED.production_units,
			updated_at      = CURRENT_TIMESTAMP;

	`

	_, err = tx.Exec(ctx, query,
		employeeID,
		salaryDate,
		branchID,
		"Present",
		amount,
		0,
		0,
	)
	if err != nil {
		return fmt.Errorf("insert/update attendance: %w", err)
	}

	//increment expense
	// Update top_sheet inside the same tx
	topSheet := &models.TopSheet{
		Date:     salaryDate,
		BranchID: branchID,
		Expense:  amount,
	}
	err = SaveTopSheetTx(tx, ctx, topSheet) // <-- must accept tx, not db
	if err != nil {
		return fmt.Errorf("save topsheet: %w", err)
	}

	//insert transaction
	transaction := &models.Transaction{
		BranchID:        branchID,
		FromID:          branchID,
		FromAccountName: "Branch",
		FromType:        "Branch",
		ToID:            employeeID,
		ToAccountName:   "",
		ToType:          "employees",
		Amount:          amount,
		TransactionType: "salary",
		CreatedAt:       salaryDate,
		Notes:           "Paying employee salary",
	}
	CreateTransactionTx(ctx, tx, transaction) // silently add transaction

	// Commit if all succeeded
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
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

// GetEmployeesNameAndIDByBranchAndRole fetches a lightweight list of active employees filtered by branch and role.
func (e *EmployeeRepo) GetEmployeesNameAndIDByBranchAndRole(ctx context.Context, branchID int64, role string) ([]*models.EmployeeNameID, error) {
	query := `
		SELECT id, name 
		FROM employees
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	// Dynamic filters
	if branchID != 0 {
		query += fmt.Sprintf(" AND branch_id = $%d", argIdx)
		args = append(args, branchID)
		argIdx++
	}

	if role != "" {
		query += fmt.Sprintf(" AND role = $%d", argIdx)
		args = append(args, role)
		argIdx++
	}

	query += " ORDER BY id ASC;"
	rows, err := e.db.Query(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, fmt.Errorf("database error: %s", pgErr.Message)
		}
		return nil, fmt.Errorf("error getting employee names and ids: %w", err)
	}
	defer rows.Close()

	list := []*models.EmployeeNameID{}
	for rows.Next() {
		var item models.EmployeeNameID
		if err := rows.Scan(&item.ID, &item.Name); err != nil {
			return nil, fmt.Errorf("error scanning employee name/id: %w", err)
		}
		list = append(list, &item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating employee name/id rows: %w", err)
	}

	return list, nil
}

// PaginatedEmployeeList returns a paginated list of employees with optional filters, dynamic sorting, or all rows if page/limit not provided.
func (e *EmployeeRepo) PaginatedEmployeeList(ctx context.Context, page, limit int, branchID int64, role, status, sortBy, sortOrder string,
) ([]*models.Employee, int, error) {

	// Base queries
	query := `SELECT id, name, role, status, mobile, email, password, passport_no, joining_date, address,
	                 base_salary, overtime_rate, branch_id, created_at, updated_at
	          FROM employees
	          WHERE 1=1`

	countQuery := `SELECT COUNT(*) FROM employees WHERE 1=1`

	args := []interface{}{}
	countArgs := []interface{}{}
	argIdx := 1

	// Dynamic filters
	if branchID != 0 {
		query += fmt.Sprintf(" AND branch_id = $%d", argIdx)
		countQuery += fmt.Sprintf(" AND branch_id = $%d", argIdx)
		args = append(args, branchID)
		countArgs = append(countArgs, branchID)
		argIdx++
	}

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

	// Dynamic sorting
	if sortBy == "" {
		sortBy = "created_at"
	}
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Only add LIMIT/OFFSET if both page and limit are provided
	if page > 0 && limit > 0 {
		offset := (page - 1) * limit
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
		args = append(args, limit, offset)
	}

	// Get total count
	var total int
	if err := e.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, 0, fmt.Errorf("database error: %s", pgErr.Message)
		}
		return nil, 0, err
	}

	// Query employees
	rows, err := e.db.Query(ctx, query, args...)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			return nil, 0, fmt.Errorf("database error: %s", pgErr.Message)
		}
		return nil, 0, err
	}
	defer rows.Close()

	employees := []*models.Employee{}
	for rows.Next() {
		var emp models.Employee
		err := rows.Scan(
			&emp.ID, &emp.Name, &emp.Role, &emp.Status,
			&emp.Mobile, &emp.Email, &emp.Password, &emp.PassportNo,
			&emp.JoiningDate, &emp.Address, &emp.BaseSalary, &emp.OvertimeRate,
			&emp.BranchID, &emp.CreatedAt, &emp.UpdatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		employees = append(employees, &emp)
	}

	return employees, total, nil
}
