package dbrepo

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

type ReportRepo struct {
	db *pgxpool.Pool
}

func NewReportRepo(db *pgxpool.Pool) *ReportRepo {
	return &ReportRepo{db: db}
}

func (r *ReportRepo) GetOrderOverView(ctx context.Context, branchID int64, summaryType string, refDate time.Time) (*models.OrderOverview, error) {
	var startDate, endDate time.Time

	switch summaryType {
	case "daily":
		startDate = refDate
		endDate = refDate
	case "weekly":
		weekday := int(refDate.Weekday())
		startDate = refDate.AddDate(0, 0, -weekday)
		endDate = startDate.AddDate(0, 0, 6)
	case "monthly":
		startDate = time.Date(refDate.Year(), refDate.Month(), 1, 0, 0, 0, 0, refDate.Location())
		endDate = startDate.AddDate(0, 1, -1)
	case "yearly":
		startDate = time.Date(refDate.Year(), 1, 1, 0, 0, 0, 0, refDate.Location())
		endDate = time.Date(refDate.Year(), 12, 31, 0, 0, 0, 0, refDate.Location())
	case "all":
		startDate = time.Time{}
		endDate = time.Now()
	default:
		return nil, fmt.Errorf("invalid summary type: %s", summaryType)
	}

	query := `
		SELECT
			COALESCE(SUM(pending), 0),
			COALESCE(SUM(checkout), 0),
			COALESCE(SUM(delivery), 0),
			COALESCE(SUM(cancelled), 0),
			COALESCE(SUM(order_count), 0)
		FROM top_sheet
		WHERE branch_id = $3
		  AND sheet_date BETWEEN $1 AND $2
	`

	var s models.OrderOverview
	err := r.db.QueryRow(ctx, query, startDate, endDate, branchID).Scan(
		&s.PendingOrders,
		&s.CheckoutOrders,
		&s.CompletedOrders,
		&s.CancelledOrders,
		&s.TotalOrders,
	)
	if err != nil {
		return nil, err
	}

	return &s, nil
}


// GetSalesPersonProgressReport gives sales progress summary for all salespersons in a branch
// grouped by day, week, month, or year — based on data from employees_progress table.
func (r *ReportRepo) GetSalesPersonProgressReport(
	ctx context.Context,
	branchID int64,
	startDate, endDate time.Time,
	reportType string,
) ([]*models.SalesPersonProgressReport, error) {

	var report []*models.SalesPersonProgressReport

	// Choose grouping format
	var dateSelect, dateGroupExpr string
	switch reportType {
	case "daily":
		dateSelect = "COALESCE(to_char(ep.sheet_date, 'YYYY-MM-DD'), '') AS date_label"
		dateGroupExpr = "to_char(ep.sheet_date, 'YYYY-MM-DD')"
	case "weekly":
		dateSelect = "COALESCE(to_char(ep.sheet_date, 'IYYY-IW'), '') AS date_label"
		dateGroupExpr = "to_char(ep.sheet_date, 'IYYY-IW')"
	case "monthly":
		dateSelect = "COALESCE(to_char(ep.sheet_date, 'YYYY-MM'), '') AS date_label"
		dateGroupExpr = "to_char(ep.sheet_date, 'YYYY-MM')"
	case "yearly":
		dateSelect = "COALESCE(to_char(ep.sheet_date, 'YYYY'), '') AS date_label"
		dateGroupExpr = "to_char(ep.sheet_date, 'YYYY')"
	default:
		return nil, fmt.Errorf("invalid reportType: %s", reportType)
	}

	groupBy := fmt.Sprintf("e.id, %s", dateGroupExpr)
	orderBy := fmt.Sprintf("%s, e.name", dateGroupExpr)

	// MAIN TABLE: employees_progress
	// LEFT JOIN employees to include missing employee info if progress exists without employee record
	query := fmt.Sprintf(`
        SELECT
            e.id AS employee_id,
            e.name,
            e.mobile,
            e.email,
            e.base_salary,
            %s,
            COALESCE(SUM(ep.sale_amount), 0)        AS total_sale_amount,
            COALESCE(SUM(ep.sale_return_amount), 0) AS total_sale_return_amount,
            COALESCE(SUM(ep.order_count), 0)        AS total_order_count,
            COALESCE(SUM(ep.item_count), 0)         AS total_item_count
        FROM employees_progress ep
        LEFT JOIN employees e 
            ON e.id = ep.employee_id
        WHERE ep.branch_id = $1
          AND ep.sheet_date BETWEEN $2 AND $3
          AND e.role = 'salesperson'
        GROUP BY %s, e.id, e.name, e.mobile, e.email, e.base_salary
        ORDER BY %s;
    `, dateSelect, groupBy, orderBy)

	rows, err := r.db.Query(ctx, query, branchID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rp models.SalesPersonProgressReport
		if err := rows.Scan(
			&rp.SalesPersonID,
			&rp.SalesPersonName,
			&rp.Mobile,
			&rp.Email,
			&rp.BaseSalary,
			&rp.Date,
			&rp.Sale,
			&rp.SaleReturn,
			&rp.OrderCount,
			&rp.ItemCount,
		); err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}
		report = append(report, &rp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return report, nil
}

// GetAllWorkersProgressReport gives a progress summary for all workers in a branch
// grouped by day, week, month, or year — based on data from employees_progress table.
func (r *ReportRepo) GetAllWorkersProgressReport(
	ctx context.Context,
	branchID int64,
	startDate, endDate time.Time,
	reportType string,
) ([]*models.WorkerProgressReport, error) {

	var report []*models.WorkerProgressReport

	// Determine grouping format based on reportType
	var dateSelect, dateGroupExpr string
	switch reportType {
	case "daily":
		dateSelect = "to_char(ep.sheet_date, 'YYYY-MM-DD') AS date_label"
		dateGroupExpr = "to_char(ep.sheet_date, 'YYYY-MM-DD')"
	case "weekly":
		dateSelect = "to_char(ep.sheet_date, 'IYYY-IW') AS date_label"
		dateGroupExpr = "to_char(ep.sheet_date, 'IYYY-IW')"
	case "monthly":
		dateSelect = "to_char(ep.sheet_date, 'YYYY-MM') AS date_label"
		dateGroupExpr = "to_char(ep.sheet_date, 'YYYY-MM')"
	case "yearly":
		dateSelect = "to_char(ep.sheet_date, 'YYYY') AS date_label"
		dateGroupExpr = "to_char(ep.sheet_date, 'YYYY')"
	default:
		return nil, fmt.Errorf("invalid reportType: %s", reportType)
	}

	// Build GROUP BY and ORDER BY dynamically
	groupBy := fmt.Sprintf("e.id, %s", dateGroupExpr)
	orderBy := fmt.Sprintf("%s, e.name", dateGroupExpr)

	// Final query: uses LEFT JOIN to include workers with no progress entries
	query := fmt.Sprintf(`
        SELECT
            e.id AS employee_id,
            e.name,
            e.mobile,
            e.email,
            e.base_salary,
            %s,
            COALESCE(SUM(ep.production_units), 0) AS total_production_units,
            COALESCE(SUM(ep.overtime_hours), 0)   AS total_overtime_hours,
            COALESCE(SUM(ep.advance_payment), 0)  AS total_advance_payment
		FROM employees_progress ep
        LEFT JOIN employees e
            ON e.id = ep.employee_id
            AND ep.sheet_date BETWEEN $2 AND $3
            AND ep.branch_id = $1
        WHERE e.branch_id = $1 AND e.role = 'worker'
        GROUP BY %s, e.id, e.name, e.mobile, e.email, e.base_salary
        ORDER BY %s;
    `, dateSelect, groupBy, orderBy)

	rows, err := r.db.Query(ctx, query, branchID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rp models.WorkerProgressReport
		if err := rows.Scan(
			&rp.WorkerID,
			&rp.WorkerName,
			&rp.Mobile,
			&rp.Email,
			&rp.BaseSalary,
			&rp.Date,
			&rp.TotalProductionUnits,
			&rp.TotalOvertimeHours,
			&rp.TotalAdvancePayment,
		); err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}
		report = append(report, &rp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return report, nil
}

// GetBranchReport gives the report of a particular employee for a given year
func (r *ReportRepo) GetBranchReport(ctx context.Context, branchID int64, startDate, endDate time.Time, reportType string) ([]*models.TopSheet, error) {
	var sheets []*models.TopSheet

	query := `
        SELECT
            id,
            sheet_date,
            branch_id,
            expense,
            cash,
            bank,
            order_count,
            delivery,
            checkout,
			cancelled
        FROM top_sheet
        WHERE branch_id = $1
          AND sheet_date >= $2
          AND sheet_date <= $3
        ORDER BY sheet_date;
    `

	rows, err := r.db.Query(ctx, query, branchID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		ts := &models.TopSheet{}
		err := rows.Scan(
			&ts.ID,
			&ts.Date,
			&ts.BranchID,
			&ts.Expense,
			&ts.Cash,
			&ts.Bank,
			&ts.OrderCount,
			&ts.Delivery,
			&ts.Checkout,
			&ts.Cancelled,
		)
		if err != nil {
			return nil, err
		}
		ts.TotalAmount = ts.Cash + ts.Bank
		ts.Balance = ts.Cash - ts.Expense
		sheets = append(sheets, ts)
	}

	return sheets, nil
}

func (r *ReportRepo) GetSalaryList(ctx context.Context, branchID, employeeID int64, startDate, endDate string) ([]*models.SalaryRecord, error) {
	query := `
		SELECT 
			ep.employee_id,
			e.name,
			e.role,
			e.base_salary,
			ep.salary,
			ep.sheet_date
		FROM employees_progress ep
		LEFT JOIN employees e ON e.id = ep.employee_id
		WHERE ep.branch_id = $1 and ep.salary > 0
	`
	args := []any{branchID}
	argPos := 2 // next placeholder index

	if employeeID != 0 {
		query += fmt.Sprintf(" AND ep.employee_id = $%d" , argPos)
		args = append(args, employeeID)
		argPos++
	}

	if startDate != "" && endDate != "" {
		query += fmt.Sprintf(" AND ep.sheet_date BETWEEN $%d AND $%d", argPos, argPos+1)
		args = append(args, startDate, endDate)
		argPos += 2
	}

	query += " ORDER BY ep.sheet_date ASC"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var salaries []*models.SalaryRecord
	for rows.Next() {
		var s models.SalaryRecord
		if err := rows.Scan(
			&s.EmployeeID,
			&s.EmployeeName,
			&s.Role,
			&s.BaseSalary,
			&s.TotalSalary,
			&s.SheetDate,
		); err != nil {
			return nil, err
		}
		salaries = append(salaries, &s)
	}
	return salaries, nil
}

