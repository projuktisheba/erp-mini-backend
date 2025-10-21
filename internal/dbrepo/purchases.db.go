package dbrepo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
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
	if p.MemoNo == "" {
		p.MemoNo = utils.GenerateMemoNo()
	}
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
	notes := "Payment for Material Purchase"
	if strings.TrimSpace(p.Notes) != "" {
		notes += p.Notes
	}

	// get the branch accounts id
	var fromAccountID int64
	err = tx.QueryRow(ctx, `
        SELECT id
        FROM accounts
		WHERE branch_id = $1 AND type = 'cash'
		LIMIT 1
    `, p.BranchID).Scan(&fromAccountID)
	if err != nil {
		return err
	}
	//insert transaction
	transaction := &models.Transaction{
		BranchID:        p.BranchID,
		MemoNo:          p.MemoNo,
		FromID:          fromAccountID,
		FromType:        "accounts",
		ToID:            p.SupplierID,
		ToType:          "suppliers",
		Amount:          p.TotalAmount,
		TransactionType: "payment",
		CreatedAt:       p.PurchaseDate,
		Notes:           notes,
	}
	_, err = CreateTransactionTx(ctx, tx, transaction) // silently add transaction
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
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

// UpdatePurchase updates an existing purchase, adjusts branch cash, top_sheet, and transaction records.
func (r *PurchaseRepo) UpdatePurchase(ctx context.Context, p *models.Purchase) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	rollback := true
	defer func() {
		if rollback {
			_ = tx.Rollback(ctx)
		}
	}()

	// Fetch old total amount for balance and expense adjustment
	var oldTotal float64
	var oldMemoNo string
	err = tx.QueryRow(ctx, `
		SELECT total_amount, memo_no
		FROM purchase
		WHERE id = $1
	`, p.ID).Scan(&oldTotal, &oldMemoNo)
	if err != nil {
		return fmt.Errorf("fetch old purchase: %w", err)
	}

	// Update purchase record
	_, err = tx.Exec(ctx, `
		UPDATE purchase
		SET 
			memo_no = $1,
			purchase_date = $2,
			supplier_id = $3,
			branch_id = $4,
			total_amount = $5,
			notes = $6,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $7
	`, p.MemoNo, p.PurchaseDate, p.SupplierID, p.BranchID, p.TotalAmount, p.Notes, p.ID)
	if err != nil {
		return fmt.Errorf("update purchase: %w", err)
	}

	// --- Adjust branch cash account based on difference ---
	diff := p.TotalAmount - oldTotal
	if diff != 0 {
		_, err = tx.Exec(ctx, `
			UPDATE accounts
			SET current_balance = current_balance - $1
			WHERE branch_id = $2 AND type = 'cash'
		`, diff, p.BranchID)
		if err != nil {
			return fmt.Errorf("update account balance: %w", err)
		}
	}

	// --- Update TopSheet (adjust expense difference) ---
	topSheet := &models.TopSheet{
		Date:     p.PurchaseDate,
		BranchID: p.BranchID,
		Expense:  diff,
	}
	if err := SaveTopSheetTx(tx, ctx, topSheet); err != nil {
		return fmt.Errorf("update topsheet expense: %w", err)
	}

	notes := "Payment adjustment for Material Purchase"
	if strings.TrimSpace(p.Notes) != "" {
		notes += " - " + p.Notes
	}

	//  transaction for adjustment
	var fromAccountID int64
	err = tx.QueryRow(ctx, `
			SELECT id FROM accounts
			WHERE branch_id = $1 AND type = 'cash'
			LIMIT 1
		`, p.BranchID).Scan(&fromAccountID)
	if err != nil {
		return fmt.Errorf("fetch account: %w", err)
	}

	transaction := &models.Transaction{
		BranchID:        p.BranchID,
		MemoNo:          p.MemoNo,
		FromID:          fromAccountID,
		FromType:        "accounts",
		ToID:            p.SupplierID,
		ToType:          "suppliers",
		Amount:          p.TotalAmount,
		TransactionType: "adjustment",
		CreatedAt:       p.PurchaseDate,
		Notes:           notes,
	}
	_, err = CreateTransactionTx(ctx, tx, transaction)
	if err != nil {
		return fmt.Errorf("create transaction: %w", err)
	}

	// Commit
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	rollback = false
	return nil
}
