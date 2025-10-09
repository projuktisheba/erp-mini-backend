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
	// Calculate start and end dates based on summary type
	var startDate, endDate time.Time

	switch summaryType {
	case "daily":
		startDate = refDate
		endDate = refDate
	case "weekly":
		weekday := int(refDate.Weekday())           // Sunday=0, Monday=1, etc.
		startDate = refDate.AddDate(0, 0, -weekday) // start of week
		endDate = startDate.AddDate(0, 0, 6)        // end of week
	case "monthly":
		startDate = time.Date(refDate.Year(), refDate.Month(), 1, 0, 0, 0, 0, refDate.Location())
		endDate = startDate.AddDate(0, 1, -1) // last day of month
	case "yearly":
		startDate = time.Date(refDate.Year(), 1, 1, 0, 0, 0, 0, refDate.Location())
		endDate = time.Date(refDate.Year(), 12, 31, 0, 0, 0, 0, refDate.Location())
	case "all":
		startDate = time.Time{} // zero time
		endDate = time.Now()
	default:
		return nil, fmt.Errorf("invalid summary type: %s", summaryType)
	}

	// SQL query with BETWEEN startDate AND endDate
	query := `
	SELECT
		COUNT(*) FILTER (WHERE status='pending'   AND branch_id=$3 AND order_date BETWEEN $1 AND $2),
		COUNT(*) FILTER (WHERE status='checkout'  AND branch_id=$3 AND order_date BETWEEN $1 AND $2),
		COUNT(*) FILTER (WHERE status='delivery'  AND branch_id=$3 AND order_date BETWEEN $1 AND $2),
		COUNT(*) FILTER (WHERE status='cancelled' AND branch_id=$3 AND order_date BETWEEN $1 AND $2),
		COUNT(*) FILTER (WHERE branch_id=$3 AND order_date BETWEEN $1 AND $2),

		COALESCE(SUM(total_payable_amount) FILTER (WHERE status='pending'   AND branch_id=$3 AND order_date BETWEEN $1 AND $2),0),
		COALESCE(SUM(total_payable_amount) FILTER (WHERE status='checkout'  AND branch_id=$3 AND order_date BETWEEN $1 AND $2),0),
		COALESCE(SUM(total_payable_amount) FILTER (WHERE status='delivery'  AND branch_id=$3 AND order_date BETWEEN $1 AND $2),0),
		COALESCE(SUM(total_payable_amount) FILTER (WHERE status='cancelled' AND branch_id=$3 AND order_date BETWEEN $1 AND $2),0),
		COALESCE(SUM(total_payable_amount) FILTER (WHERE branch_id=$3 AND order_date BETWEEN $1 AND $2),0)
	FROM orders

	`

	var s models.OrderOverview
	err := r.db.QueryRow(ctx, query, startDate, endDate, branchID).Scan(
		&s.PendingOrders,
		&s.CheckoutOrders,
		&s.CompletedOrders,
		&s.CancelledOrders,
		&s.TotalOrders,

		&s.PendingOrdersAmount,
		&s.CheckoutOrdersAmount,
		&s.CompletedOrdersAmount,
		&s.CancelledOrdersAmount,
		&s.TotalOrdersAmount,
	)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

// GetEmployeeProgressReport gives the progress report of a particular employee for a given period
// GetSalesPersonProgressReport gives the sales progress report for all salespersons in a branch
func (r *ReportRepo) GetSalesPersonProgressReport(
	ctx context.Context,
	branchID int64,
	startDate, endDate time.Time,
	reportType string,
) ([]*models.SalesPersonProgressReport, error) {

	var report []*models.SalesPersonProgressReport

	// Choose date grouping
	var dateSelect, groupBy, orderBy string

	switch reportType {
	case "daily":
		dateSelect = "to_char(o.order_date, 'YYYY-MM-DD') AS date_label"
		groupBy = "s.id, to_char(o.order_date, 'YYYY-MM-DD'), p.product_name"
		orderBy = "s.name, date_label, p.product_name"

	case "weekly":
		dateSelect = "to_char(o.order_date, 'IYYY-IW') AS date_label"
		groupBy = "s.id, to_char(o.order_date, 'IYYY-IW'), p.product_name"
		orderBy = "s.name, date_label, p.product_name"

	case "monthly":
		dateSelect = "to_char(o.order_date, 'YYYY-MM') AS date_label"
		groupBy = "s.id, to_char(o.order_date, 'YYYY-MM'), p.product_name"
		orderBy = "s.name, date_label, p.product_name"

	case "yearly":
		dateSelect = "to_char(o.order_date, 'YYYY') AS date_label"
		groupBy = "s.id, to_char(o.order_date, 'YYYY'), p.product_name"
		orderBy = "s.name, date_label, p.product_name"

	default:
		return nil, fmt.Errorf("invalid reportType: %s", reportType)
	}

	// Include salesperson info
	query := fmt.Sprintf(`
        SELECT
            s.id AS salesperson_id,
            s.name,
            s.mobile,
            s.email,
            s.base_salary,
            %s,
            p.product_name,
            COUNT(DISTINCT o.id) AS order_count,
            COALESCE(SUM(oi.quantity), 0) AS item_count,
            COALESCE(SUM(oi.subtotal), 0) AS total_sale,
            COALESCE(SUM(oi.subtotal) FILTER (WHERE o.status = 'cancelled'), 0) AS total_sale_return
        FROM employees s
        LEFT JOIN orders o 
               ON s.id = o.salesperson_id 
              AND o.order_date >= $2 
              AND o.order_date <= $3
        LEFT JOIN order_items oi ON oi.memo_no = o.memo_no
        LEFT JOIN products p     ON p.id = oi.product_id
        WHERE s.branch_id = $1
          AND s.role = 'salesperson'
          AND o.order_date IS NOT NULL
        GROUP BY %s, s.id, s.name, s.mobile, s.email, s.base_salary
        ORDER BY %s;
    `, dateSelect, groupBy, orderBy)

	rows, err := r.db.Query(ctx, query, branchID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		rp := &models.SalesPersonProgressReport{}
		err := rows.Scan(
			&rp.SalesPersonID,
			&rp.SalesPersonName,
			&rp.Mobile,
			&rp.Email,
			&rp.BaseSalary,
			&rp.Date,
			&rp.ProductName,
			&rp.OrderCount,
			&rp.ItemCount,
			&rp.Sale,
			&rp.SaleReturn,
		)
		if err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}
		report = append(report, rp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return report, nil
}

// GetAllWorkersProgressReport gives attendance/progress summary for all employees in a branch
func (r *ReportRepo) GetAllWorkersProgressReport(
	ctx context.Context,
	branchID int64,
	startDate, endDate time.Time,
	reportType string,
) ([]*models.WorkerProgressReport, error) {

	var report []*models.WorkerProgressReport

	// Choose date grouping
	var dateSelect, dateGroupExpr, groupBy, orderBy string
	switch reportType {
	case "daily":
		dateSelect = "to_char(a.work_date, 'YYYY-MM-DD') AS date_label"
		dateGroupExpr = "to_char(a.work_date, 'YYYY-MM-DD')"
	case "weekly":
		dateSelect = "to_char(a.work_date, 'IYYY-IW') AS date_label"
		dateGroupExpr = "to_char(a.work_date, 'IYYY-IW')"
	case "monthly":
		dateSelect = "to_char(a.work_date, 'YYYY-MM') AS date_label"
		dateGroupExpr = "to_char(a.work_date, 'YYYY-MM')"
	case "yearly":
		dateSelect = "to_char(a.work_date, 'YYYY') AS date_label"
		dateGroupExpr = "to_char(a.work_date, 'YYYY')"
	default:
		return nil, fmt.Errorf("invalid reportType: %s", reportType)
	}

	groupBy = fmt.Sprintf("e.id, %s", dateGroupExpr)
	orderBy = fmt.Sprintf("e.name, %s", dateGroupExpr)

	// Final query (excluding rows with NULL date)
	query := fmt.Sprintf(`
        SELECT
            e.id AS employee_id,
            e.name,
            e.mobile,
            e.email,
            e.base_salary,
            %s,
            COUNT(a.id) AS present_days,
            COALESCE(SUM(a.advance_payment), 0)  AS total_advance_payment,
            COALESCE(SUM(a.production_units), 0) AS total_production_units,
            COALESCE(SUM(a.overtime_hours), 0)   AS total_overtime_hours
        FROM employees e
        LEFT JOIN attendance a 
            ON e.id = a.employee_id 
            AND a.work_date BETWEEN $2 AND $3
        WHERE e.branch_id = $1 AND e.role = 'worker'
          AND a.work_date IS NOT NULL
        GROUP BY %s, e.id, e.name, e.mobile, e.email, e.base_salary
        ORDER BY %s;
    `, dateSelect, groupBy, orderBy)

	rows, err := r.db.Query(ctx, query, branchID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		rp := &models.WorkerProgressReport{}
		err := rows.Scan(
			&rp.WorkerID,
			&rp.WorkerName,
			&rp.Mobile,
			&rp.Email,
			&rp.BaseSalary,
			&rp.Date,
			&rp.PresentDays,
			&rp.TotalAdvancePayment,
			&rp.TotalProductionUnits,
			&rp.TotalOvertimeHours,
		)
		if err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}
		report = append(report, rp)
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
            checkout
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
