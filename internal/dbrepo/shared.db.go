package dbrepo

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

func SaveTopSheet(db *pgxpool.Pool, ctx context.Context, ts *models.TopSheet) error {
	query := `
        INSERT INTO top_sheet (
            sheet_date, branch_id, expense, cash, bank, order_count, delivery, checkout,ready_made
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8, $9)
        ON CONFLICT (sheet_date, branch_id) DO UPDATE SET
            expense      = top_sheet.expense + EXCLUDED.expense,
            cash         = top_sheet.cash + EXCLUDED.cash,
            bank         = top_sheet.bank + EXCLUDED.bank,
            order_count  = top_sheet.order_count + EXCLUDED.order_count,
            delivery     = top_sheet.delivery + EXCLUDED.delivery,
            checkout     = top_sheet.checkout + EXCLUDED.checkout,
            ready_made     = top_sheet.ready_made + EXCLUDED.ready_made;
    `
	_, err := db.Exec(ctx, query, ts.Date, ts.BranchID, ts.Expense, ts.Cash, ts.Bank, ts.OrderCount, ts.Delivery, ts.Checkout, ts.ReadyMade)
	return err
}
func SaveTopSheetTx(tx pgx.Tx, ctx context.Context, ts *models.TopSheet) error {
	query := `
		INSERT INTO top_sheet (
			sheet_date,
			branch_id,
			expense,
			cash,
			bank,
			order_count,
			pending,
			delivery,
			checkout,
			cancelled,
			ready_made
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (sheet_date, branch_id) DO UPDATE SET
			expense      = top_sheet.expense + EXCLUDED.expense,
			cash         = top_sheet.cash + EXCLUDED.cash,
			bank         = top_sheet.bank + EXCLUDED.bank,
			order_count  = top_sheet.order_count + EXCLUDED.order_count,
			pending      = top_sheet.pending + EXCLUDED.pending,
			delivery     = top_sheet.delivery + EXCLUDED.delivery,
			checkout     = top_sheet.checkout + EXCLUDED.checkout,
			cancelled    = top_sheet.cancelled + EXCLUDED.cancelled,
			ready_made   = top_sheet.ready_made + EXCLUDED.ready_made;
	`

	_, err := tx.Exec(
		ctx,
		query,
		ts.Date,
		ts.BranchID,
		ts.Expense,
		ts.Cash,
		ts.Bank,
		ts.OrderCount,
		ts.Pending,
		ts.Delivery,
		ts.Checkout,
		ts.Cancelled,
		ts.ReadyMade,
	)
	return err
}


// UpdateSalespersonProgressReportTx updates or inserts salesperson progress
func UpdateSalespersonProgressReportTx(tx pgx.Tx, ctx context.Context, ts *models.SalespersonProgress) error {
	query := `
	INSERT INTO employees_progress (
		sheet_date, branch_id, employee_id,
		sale_amount, sale_return_amount,
		order_count, salary
	) VALUES ($1,$2,$3,$4,$5,$6,$7)
	ON CONFLICT (sheet_date, employee_id) DO UPDATE SET
		sale_amount        = employees_progress.sale_amount + EXCLUDED.sale_amount,
		sale_return_amount = employees_progress.sale_return_amount + EXCLUDED.sale_return_amount,
		order_count        = employees_progress.order_count + EXCLUDED.order_count;
	`
	_, err := tx.Exec(ctx, query,
		ts.Date,
		ts.BranchID,
		ts.EmployeeID,
		ts.SaleAmount,
		ts.SaleReturnAmount,
		ts.OrderCount,
		ts.Salary,
	)
	return err
}

// UpdateWorkerProgressReportTx updates or inserts worker progress
func UpdateWorkerProgressReportTx(tx pgx.Tx, ctx context.Context, wp *models.WorkerProgress) error {
	query := `
	INSERT INTO employees_progress (
		sheet_date, branch_id, employee_id,
		production_units, overtime_hours,
		advance_payment, salary
	) VALUES ($1,$2,$3,$4,$5,$6,$7)
	ON CONFLICT (sheet_date, employee_id) DO UPDATE SET
		production_units = employees_progress.production_units + EXCLUDED.production_units,
		overtime_hours   = employees_progress.overtime_hours + EXCLUDED.overtime_hours,
		advance_payment  = employees_progress.advance_payment + EXCLUDED.advance_payment;
	`
	_, err := tx.Exec(ctx, query,
		wp.Date,
		wp.BranchID,
		wp.EmployeeID,
		wp.ProductionUnits,
		wp.OvertimeHours,
		wp.AdvancePayment,
		wp.Salary,
	)
	return err
}
// SubmitEmployeeSalaryTx updates or inserts employee salary
func SubmitEmployeeSalaryTx(tx pgx.Tx, ctx context.Context, date time.Time, branchID int64, employeeID int64, salary float64) error {
	query := `
	INSERT INTO employees_progress (
		sheet_date, branch_id, employee_id, salary
	) VALUES ($1,$2,$3,$4)
	ON CONFLICT (sheet_date, employee_id) DO UPDATE SET
		salary  = employees_progress.salary + EXCLUDED.salary;
	`
	_, err := tx.Exec(ctx, query,
		date,
		branchID,
		employeeID,
		salary,
	)
	return err
}