package db

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

func (store *Store) execTx(ctx context.Context, callback func(*Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)

	if err != nil {
		return err
	}

	queries := New(tx)

	if err = callback(queries); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("Transaction error: %v, Rollback error: %v", err, rbErr)
		}

		return err
	}

	return tx.Commit()
}

type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"fromAccount"`
	ToAccount   Account  `json:"toAccount"`
	FromEntry   Entry    `json:"fromEntry"`
	ToEntry     Entry    `json:"toEntry"`
}

func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(q *Queries) error {
		var err error

		result.Transfer, err = q.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})

		if err != nil {
			return err
		}

		result.FromEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})

		if err != nil {
			return err
		}

		result.ToEntry, err = q.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    arg.Amount,
		})

		if err != nil {
			return err
		}

		originalFromAccount, err := q.GetAccountForUpdate(ctx, arg.FromAccountID)

		if err != nil {
			return err
		}

		result.FromAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
			ID:      arg.FromAccountID,
			Balance: originalFromAccount.Balance - arg.Amount,
		})

		if err != nil {
			return err
		}

		originalToAccount, err := q.GetAccountForUpdate(ctx, arg.ToAccountID)

		if err != nil {
			return err
		}

		result.ToAccount, err = q.UpdateAccount(ctx, UpdateAccountParams{
			ID:      originalToAccount.ID,
			Balance: originalToAccount.Balance + arg.Amount,
		})

		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}
