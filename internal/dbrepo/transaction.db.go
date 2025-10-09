package dbrepo

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{db: db}
}

// CreateTransaction inserts a new transaction into the database
func (r *TransactionRepo) CreateTransaction(ctx context.Context, t *models.Transaction) (int64, error) {
	var transactionID int64
	err := r.db.QueryRow(
		ctx,
		`INSERT INTO transactions 
			(branch_id, from_entity_id, from_entity_type, to_entity_id, to_entity_type, amount, transaction_type, notes, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		 RETURNING transaction_id`,
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

// CreateTransaction inserts a new transaction into the database using a pgx transaction
func CreateTransactionTx(ctx context.Context, tx pgx.Tx, t *models.Transaction) (int64, error) {
	var transactionID int64

	query := `
		INSERT INTO transactions 
			(branch_id, from_entity_id, from_entity_type, to_entity_id, to_entity_type, amount, transaction_type, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
		RETURNING transaction_id
	`

	err := tx.QueryRow(
		ctx,
		query,
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

func (r *TransactionRepo) GetTransactionSummary(
	ctx context.Context,
	branchID int64,
	startDate, endDate string,
	fromID, toID *int64,
	fromType, toType, trxType *string,
) ([]*models.Transaction, error) {

	query := `
		SELECT id, transaction_id, branch_id, from_id, from_type, to_id, to_type, amount, transaction_type, created_at, notes
		FROM transactions
		WHERE created_at::date BETWEEN $1 AND $2
	`
	args := []interface{}{startDate, endDate}
	argID := 3
	if branchID > 0 {
		query += ` AND branch_id=$` + strconv.Itoa(argID)
		args = append(args, *fromID)
		argID++
	}
	if fromID != nil {
		query += ` AND from_id=$` + strconv.Itoa(argID)
		args = append(args, *fromID)
		argID++
	}
	if toID != nil {
		query += ` AND to_id=$` + strconv.Itoa(argID)
		args = append(args, *toID)
		argID++
	}
	if fromType != nil {
		query += ` AND from_type=$` + strconv.Itoa(argID)
		args = append(args, *fromType)
		argID++
	}
	if toType != nil {
		query += ` AND to_type=$` + strconv.Itoa(argID)
		args = append(args, *toType)
		argID++
	}
	if trxType != nil {
		query += ` AND transaction_type=$` + strconv.Itoa(argID)
		args = append(args, *trxType)
		argID++
	}

	query += ` ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []*models.Transaction
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(
			&t.ID, &t.TransactionID, &t.BranchID, &t.BranchID, &t.FromID, &t.FromType, &t.ToID, &t.ToType,
			&t.Amount, &t.TransactionType, &t.CreatedAt, &t.Notes,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, &t)
	}

	return transactions, nil
}
func (r *TransactionRepo) ListTransactionsPaginated(
	ctx context.Context,
	branchID int64,
	pageNo, pageLength int,
	fromID, toID *int64,
	fromType, toType, trxType *string,
) ([]*models.Transaction, error) {

	query := `
		SELECT id, transaction_id, branch_id, from_id, from_type, to_id, to_type, amount, transaction_type, created_at, notes
		FROM transactions
		WHERE 1=1
	`
	args := []interface{}{}
	argID := 1
	if branchID > 0 {
		query += ` AND branch_id=$` + strconv.Itoa(argID)
		args = append(args, *fromID)
		argID++
	}
	if fromID != nil {
		query += ` AND from_id=$` + strconv.Itoa(argID)
		args = append(args, *fromID)
		argID++
	}
	if toID != nil {
		query += ` AND to_id=$` + strconv.Itoa(argID)
		args = append(args, *toID)
		argID++
	}
	if fromType != nil {
		query += ` AND from_type=$` + strconv.Itoa(argID)
		args = append(args, *fromType)
		argID++
	}
	if toType != nil {
		query += ` AND to_type=$` + strconv.Itoa(argID)
		args = append(args, *toType)
		argID++
	}
	if trxType != nil {
		query += ` AND transaction_type=$` + strconv.Itoa(argID)
		args = append(args, *trxType)
		argID++
	}

	query += ` ORDER BY created_at DESC`

	if pageLength != -1 {
		offset := (pageNo - 1) * pageLength
		query += ` LIMIT $` + strconv.Itoa(argID) + ` OFFSET $` + strconv.Itoa(argID+1)
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
			&t.ID, &t.TransactionID, &t.BranchID, &t.FromID, &t.FromType, &t.ToID, &t.ToType,
			&t.Amount, &t.TransactionType, &t.CreatedAt, &t.Notes,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, &t)
	}

	return transactions, nil
}
