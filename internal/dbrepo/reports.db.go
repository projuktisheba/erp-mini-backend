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

func (r *ReportRepo) GetOrderOverView(ctx context.Context, summaryType string, refDate time.Time) (*models.OrderOverview, error) {
	// Calculate start and end dates based on summary type
	var startDate, endDate time.Time

	switch summaryType {
	case "daily":
		startDate = refDate
		endDate = refDate
	case "weekly":
		weekday := int(refDate.Weekday())          // Sunday=0, Monday=1, etc.
		startDate = refDate.AddDate(0, 0, -weekday) // start of week
		endDate = startDate.AddDate(0, 0, 6)        // end of week
	case "monthly":
		startDate = time.Date(refDate.Year(), refDate.Month(), 1, 0, 0, 0, 0, refDate.Location())
		endDate = startDate.AddDate(0, 1, -1)       // last day of month
	case "yearly":
		startDate = time.Date(refDate.Year(), 1, 1, 0, 0, 0, 0, refDate.Location())
		endDate = time.Date(refDate.Year(), 12, 31, 0, 0, 0, 0, refDate.Location())
	case "all":
		startDate = time.Time{}  // zero time
		endDate = time.Now()
	default:
		return nil, fmt.Errorf("invalid summary type: %s", summaryType)
	}

	// SQL query with BETWEEN startDate AND endDate
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status='pending'   AND order_date BETWEEN $1 AND $2),
			COUNT(*) FILTER (WHERE status='checkout'  AND order_date BETWEEN $1 AND $2),
			COUNT(*) FILTER (WHERE status='delivery'  AND order_date BETWEEN $1 AND $2),
			COUNT(*) FILTER (WHERE status='cancelled' AND order_date BETWEEN $1 AND $2),
			COUNT(*) FILTER (WHERE order_date BETWEEN $1 AND $2),

			COALESCE(SUM(total_payable_amount) FILTER (WHERE status='pending'   AND order_date BETWEEN $1 AND $2),0),
			COALESCE(SUM(total_payable_amount) FILTER (WHERE status='checkout'  AND order_date BETWEEN $1 AND $2),0),
			COALESCE(SUM(total_payable_amount) FILTER (WHERE status='delivery'  AND order_date BETWEEN $1 AND $2),0),
			COALESCE(SUM(total_payable_amount) FILTER (WHERE status='cancelled' AND order_date BETWEEN $1 AND $2),0),
			COALESCE(SUM(total_payable_amount) FILTER (WHERE order_date BETWEEN $1 AND $2),0)
		FROM orders
	`

	var s models.OrderOverview
	err := r.db.QueryRow(ctx, query, startDate, endDate).Scan(
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
