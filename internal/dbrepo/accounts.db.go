package dbrepo

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/projuktisheba/erp-mini-api/internal/models"
)

type AccountRepo struct {
	db *pgxpool.Pool
}

func NewAccountRepo(db *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{db: db}
}

func (a *AccountRepo)GetAccounts(ctx context.Context) ([]*models.Account, error) {
	rows, err := a.db.Query(ctx, `
        SELECT id, name, type, current_balance, created_at, updated_at
        FROM accounts
        ORDER BY id
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.Account
	for rows.Next() {
		var a models.Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.CurrentBalance, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, &a)
	}

	return accounts, nil
}

func (a *AccountRepo)GetAccountsNames(ctx context.Context) ([]*models.AccountNameID, error) {
	rows, err := a.db.Query(ctx, `
        SELECT id, name
        FROM accounts
        ORDER BY id
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.AccountNameID
	for rows.Next() {
		var a models.AccountNameID
		if err := rows.Scan(&a.ID, &a.Name); err != nil {
			return nil, err
		}
		accounts = append(accounts, &a)
	}

	return accounts, nil
}
