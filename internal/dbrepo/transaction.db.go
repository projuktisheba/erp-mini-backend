package dbrepo

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{db: db}
}

func (r *TransactionRepo) GetTransactionSummary(
	ctx context.Context,
	startDate, endDate string,
	fromID, toID *int64,
	fromType, toType, trxType *string,
) ([]*models.Transaction, error) {

	query := `
		SELECT id, transaction_id, from_id, from_type, to_id, to_type, amount, transaction_type, created_at, notes
		FROM transactions
		WHERE created_at::date BETWEEN $1 AND $2
	`
	args := []interface{}{startDate, endDate}
	argID := 3

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
			&t.ID, &t.TransactionID, &t.FromID, &t.FromType, &t.ToID, &t.ToType,
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
	pageNo, pageLength int,
	fromID, toID *int64,
	fromType, toType, trxType *string,
) ([]*models.Transaction, error) {

	query := `
		SELECT id, transaction_id, from_id, from_type, to_id, to_type, amount, transaction_type, created_at, notes
		FROM transactions
		WHERE 1=1
	`
	args := []interface{}{}
	argID := 1

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
			&t.ID, &t.TransactionID, &t.FromID, &t.FromType, &t.ToID, &t.ToType,
			&t.Amount, &t.TransactionType, &t.CreatedAt, &t.Notes,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, &t)
	}

	return transactions, nil
}
