package dbrepo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

type PurchaseRepo struct {
	db *pgxpool.Pool
}

func NewPurchaseRepo(db *pgxpool.Pool) *PurchaseRepo {
	return &PurchaseRepo{db: db}
}

// CreatePurchase inserts a new purchase, updates the branch cash account, and logs expense in top_sheet
func (r *PurchaseRepo) CreatePurchase(ctx context.Context, p *models.Purchase) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Ensure rollback on early return
	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback(ctx)
		}
	}()

	// Insert purchase
	query := `
		INSERT INTO purchase 
		(memo_no, purchase_date, supplier_id, branch_id, total_amount, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRow(ctx, query,
		p.MemoNo,
		p.PurchaseDate,
		p.SupplierID,
		p.BranchID,
		p.TotalAmount,
		p.Notes,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert purchase: %w", err)
	}

	// Update branch cash account
	_, err = tx.Exec(ctx, `
		UPDATE accounts
		SET current_balance = current_balance - $1
		WHERE branch_id = $2 AND type='cash'
	`, p.TotalAmount, p.BranchID)
	if err != nil {
		return fmt.Errorf("update account balance: %w", err)
	}

	// --- Update TopSheet: increase expense ---
	topSheet := &models.TopSheet{
		Date:     p.PurchaseDate,
		BranchID: p.BranchID,
		Expense:  p.TotalAmount,
	}
	if err := SaveTopSheetTx(tx, ctx, topSheet); err != nil {
		return fmt.Errorf("update topsheet expense: %w", err)
	}
	notes := fmt.Sprintf("Payment for Material Purchase || %s", p.Notes)

	_, err = tx.Exec(ctx, `
		INSERT INTO transactions (
			from_entity_id,
			from_entity_type,
			to_entity_id,
			to_entity_type,
			amount,
			transaction_type,
			notes,
			branch_id
		)
		VALUES ($1, 'branches', $2, 'suppliers', $3, 'payment', $4, $5)
	`, p.BranchID, p.SupplierID, p.TotalAmount, notes, p.BranchID)
	if err != nil {
		return err
	}
	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	rollback = false // commit succeeded, no rollback needed

	return nil
}

// ListPurchasesPaginated lists purchases with dynamic filters and pagination
func (r *PurchaseRepo) ListPurchasesPaginated(
	ctx context.Context,
	memoNo string,
	supplierID, branchID int64,
	fromDate, toDate *time.Time,
	pageNo, pageLen int,
) ([]*models.Purchase, int, error) {

	// Base SELECT query
	baseQuery := `
		SELECT 
			p.id, 
			p.memo_no, 
			p.purchase_date, 
			p.supplier_id,
			s.name AS supplier_name, 
			p.branch_id, 
			p.total_amount, 
			p.notes, 
			p.created_at, 
			p.updated_at
		FROM purchase p
		JOIN suppliers s ON p.supplier_id = s.id
	`

	// Build dynamic WHERE conditions
	var conditions []string
	var args []interface{}
	argPos := 1

	if memoNo != "" {
		conditions = append(conditions, fmt.Sprintf("p.memo_no ILIKE '%%' || $%d || '%%'", argPos))
		args = append(args, memoNo)
		argPos++
	}
	if supplierID > 0 {
		conditions = append(conditions, fmt.Sprintf("p.supplier_id = $%d", argPos))
		args = append(args, supplierID)
		argPos++
	}
	if branchID > 0 {
		conditions = append(conditions, fmt.Sprintf("p.branch_id = $%d", argPos))
		args = append(args, branchID)
		argPos++
	}
	if fromDate != nil {
		conditions = append(conditions, fmt.Sprintf("p.purchase_date >= $%d", argPos))
		args = append(args, *fromDate)
		argPos++
	}
	if toDate != nil {
		conditions = append(conditions, fmt.Sprintf("p.purchase_date <= $%d", argPos))
		args = append(args, *toDate)
		argPos++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	// COUNT query for total rows
	countQuery := "SELECT COUNT(*) FROM purchase p JOIN suppliers s ON p.supplier_id = s.id" + whereClause
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count purchases: %w", err)
	}

	query := baseQuery + whereClause + " ORDER BY p.purchase_date DESC"
	// Add ORDER, LIMIT, OFFSET for pagination
	if pageLen > 0 && pageNo > 0 {
		offset := (pageNo - 1) * pageLen

		query += fmt.Sprintf(" LIMIT %d OFFSET %d", pageLen, offset)
	}

	// Execute final query
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list purchases: %w", err)
	}
	defer rows.Close()

	var purchases []*models.Purchase
	for rows.Next() {
		p := &models.Purchase{}
		if err := rows.Scan(
			&p.ID,
			&p.MemoNo,
			&p.PurchaseDate,
			&p.SupplierID,
			&p.SupplierName,
			&p.BranchID,
			&p.TotalAmount,
			&p.Notes,
			&p.CreatedAt,
			&p.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan purchase: %w", err)
		}
		purchases = append(purchases, p)
	}

	return purchases, total, nil
}
