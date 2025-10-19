package dbrepo

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
	"github.com/projuktisheba/erp-mini-api/internal/utils"
)

type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{db: db}
}

// CreateTransaction inserts a new transaction
func (r *TransactionRepo) CreateTransaction(ctx context.Context, t *models.Transaction) (int64, error) {
	// generate memo_no if empty
	if t.MemoNo == "" {
		t.MemoNo = utils.GenerateMemoNo()
	}

	var transactionID int64
	query := `
		INSERT INTO transactions
			(memo_no, branch_id, from_entity_id, from_entity_type, to_entity_id, to_entity_type, amount, transaction_type, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		RETURNING transaction_id
	`
	err := r.db.QueryRow(ctx, query,
		t.MemoNo,
		t.BranchID,
		t.FromID,
		t.FromType,
		t.ToID,
		t.ToType,
		t.Amount,
		t.TransactionType,
		t.Notes,
	).Scan(&transactionID)

	if err != nil {
		return 0, fmt.Errorf("failed to create transaction: %w", err)
	}

	return transactionID, nil
}

// CreateTransactionTx inserts a transaction within an existing tx
func CreateTransactionTx(ctx context.Context, tx pgx.Tx, t *models.Transaction) (int64, error) {
	if t.MemoNo == "" {
		t.MemoNo = utils.GenerateMemoNo()
	}

	var transactionID int64
	query := `
		INSERT INTO transactions
			(memo_no, branch_id, from_entity_id, from_entity_type, to_entity_id, to_entity_type, amount, transaction_type, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		RETURNING transaction_id
	`
	err := tx.QueryRow(ctx, query,
		t.MemoNo,
		t.BranchID,
		t.FromID,
		t.FromType,
		t.ToID,
		t.ToType,
		t.Amount,
		t.TransactionType,
		t.Notes,
	).Scan(&transactionID)

	if err != nil {
		return 0, fmt.Errorf("failed to create transaction in tx: %w", err)
	}

	return transactionID, nil
}

// ListTransactionsPaginated retrieves transactions with optional filters
func (r *TransactionRepo) ListTransactionsPaginated(ctx context.Context, memo string, branchID int64, pageNo, pageLength int, fromID, toID *int64, fromType, toType, trxType *string) ([]*models.Transaction, error) {

	query := `
		SELECT transaction_id, memo_no, branch_id, from_entity_id, from_entity_type, to_entity_id, to_entity_type,
		       amount, transaction_type, notes, created_at
		FROM transactions
		WHERE 1=1
	`
	args := []interface{}{}
	argID := 1

	if memo != "" {
		query += ` AND memo_no=$` + fmt.Sprint(argID)
		args = append(args, memo)
		argID++
	}
	if branchID > 0 {
		query += ` AND branch_id=$` + fmt.Sprint(argID)
		args = append(args, branchID)
		argID++
	}
	if fromID != nil {
		query += ` AND from_entity_id=$` + fmt.Sprint(argID)
		args = append(args, *fromID)
		argID++
	}
	if toID != nil {
		query += ` AND to_entity_id=$` + fmt.Sprint(argID)
		args = append(args, *toID)
		argID++
	}
	if fromType != nil {
		query += ` AND from_entity_type=$` + fmt.Sprint(argID)
		args = append(args, *fromType)
		argID++
	}
	if toType != nil {
		query += ` AND to_entity_type=$` + fmt.Sprint(argID)
		args = append(args, *toType)
		argID++
	}
	if trxType != nil {
		query += ` AND transaction_type=$` + fmt.Sprint(argID)
		args = append(args, *trxType)
		argID++
	}

	query += ` ORDER BY created_at DESC`

	if pageLength > 0 {
		offset := (pageNo - 1) * pageLength
		query += ` LIMIT $` + fmt.Sprint(argID) + ` OFFSET $` + fmt.Sprint(argID+1)
		args = append(args, pageLength, offset)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(
			&t.TransactionID, &t.MemoNo, &t.BranchID,
			&t.FromID, &t.FromType, &t.ToID, &t.ToType,
			&t.Amount, &t.TransactionType, &t.Notes, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, &t)
	}

	return transactions, nil
}

func (r *TransactionRepo) GetTransactionSummary(
	ctx context.Context,
	branchID int64,
	startDate, endDate string,
	trxType *string,
) ([]*models.Transaction, error) {

	// Base query
	query := `
	SELECT
		t.transaction_id,
		t.memo_no,
		t.branch_id,
		t.from_entity_id,
		t.from_entity_type,
		COALESCE(
			CASE 
				WHEN t.from_entity_type = 'accounts' THEN a1.name
				WHEN t.from_entity_type = 'customers' THEN c1.name
				WHEN t.from_entity_type = 'employees' THEN e1.name
				WHEN t.from_entity_type = 'suppliers' THEN s1.name
				ELSE NULL
			END, '-') AS from_entity_name,
		t.to_entity_id,
		t.to_entity_type,
		COALESCE(
			CASE 
				WHEN t.to_entity_type = 'accounts' THEN a2.name
				WHEN t.to_entity_type = 'customers' THEN c2.name
				WHEN t.to_entity_type = 'employees' THEN e2.name
				WHEN t.to_entity_type = 'suppliers' THEN s2.name
				ELSE NULL
			END, '-') AS to_entity_name,
		t.amount,
		t.transaction_type,
		t.notes,
		t.created_at
	FROM transactions t
	LEFT JOIN accounts a1 ON t.from_entity_type = 'accounts' AND t.from_entity_id = a1.id
	LEFT JOIN customers c1 ON t.from_entity_type = 'customers' AND t.from_entity_id = c1.id
	LEFT JOIN employees e1 ON t.from_entity_type = 'employees' AND t.from_entity_id = e1.id
	LEFT JOIN suppliers s1 ON t.from_entity_type = 'suppliers' AND t.from_entity_id = s1.id
	LEFT JOIN accounts a2 ON t.to_entity_type = 'accounts' AND t.to_entity_id = a2.id
	LEFT JOIN customers c2 ON t.to_entity_type = 'customers' AND t.to_entity_id = c2.id
	LEFT JOIN employees e2 ON t.to_entity_type = 'employees' AND t.to_entity_id = e2.id
	LEFT JOIN suppliers s2 ON t.to_entity_type = 'suppliers' AND t.to_entity_id = s2.id
	`

	// Dynamic WHERE conditions
	whereClauses := []string{"t.created_at::date BETWEEN $1 AND $2"}
	args := []interface{}{startDate, endDate}
	argID := 3

	if branchID > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("t.branch_id=$%d", argID))
		args = append(args, branchID)
		argID++
	}

	if trxType != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("t.transaction_type=$%d", argID))
		args = append(args, *trxType)
		argID++
	}

	// Combine WHERE clauses
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Add ORDER BY
	query += " ORDER BY t.created_at DESC"

	// Execute query
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Scan results
	var summaries []*models.Transaction
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(
			&t.TransactionID, &t.MemoNo, &t.BranchID,
			&t.FromID, &t.FromType, &t.FromAccountName,
			&t.ToID, &t.ToType, &t.ToAccountName,
			&t.Amount, &t.TransactionType, &t.Notes, &t.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("row scan failed: %w", err)
		}
		summaries = append(summaries, &t)
	}

	return summaries, nil
}
