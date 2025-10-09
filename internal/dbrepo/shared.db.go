package dbrepo

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

func SaveTopSheet(db *pgxpool.Pool, ctx context.Context, ts *models.TopSheet) error {
	query := `
        INSERT INTO top_sheet (
            sheet_date, branch_id, expense, cash, bank, order_count, delivery, checkout
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        ON CONFLICT (sheet_date, branch_id) DO UPDATE SET
            expense      = top_sheet.expense + EXCLUDED.expense,
            cash         = top_sheet.cash + EXCLUDED.cash,
            bank         = top_sheet.bank + EXCLUDED.bank,
            order_count  = top_sheet.order_count + EXCLUDED.order_count,
            delivery     = top_sheet.delivery + EXCLUDED.delivery,
            checkout     = top_sheet.checkout + EXCLUDED.checkout;
    `
	_, err := db.Exec(ctx, query, ts.Date, ts.BranchID, ts.Expense, ts.Cash, ts.Bank, ts.OrderCount, ts.Delivery, ts.Checkout)
	return err
}
func SaveTopSheetTx(tx pgx.Tx, ctx context.Context, ts *models.TopSheet) error {
	query := `
        INSERT INTO top_sheet (
            sheet_date, branch_id, expense, cash, bank, order_count, delivery, checkout
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
        ON CONFLICT (sheet_date, branch_id) DO UPDATE SET
            expense      = top_sheet.expense + EXCLUDED.expense,
            cash         = top_sheet.cash + EXCLUDED.cash,
            bank         = top_sheet.bank + EXCLUDED.bank,
            order_count  = top_sheet.order_count + EXCLUDED.order_count,
            delivery     = top_sheet.delivery + EXCLUDED.delivery,
            checkout     = top_sheet.checkout + EXCLUDED.checkout;
    `
	_, err := tx.Exec(ctx, query, ts.Date, ts.BranchID, ts.Expense, ts.Cash, ts.Bank, ts.OrderCount, ts.Delivery, ts.Checkout)
	return err
}
