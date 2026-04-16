package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

type Store struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		Queries: New(db),
		db:      db,
	}
}

func (store *Store) execTxn(ctx context.Context, fn func(*Queries) error) error {
	txn, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	queries := New(txn)
	err = fn(queries)
	if err != nil {
		rollbackErr := txn.Rollback()
		if rollbackErr != nil {
			return fmt.Errorf("Transaction error: %v, Rollback error: %v", err, rollbackErr)
		}

		return err
	}

	return txn.Commit()
}

type TransferTxnRes struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

func (store *Store) TransferTxn(ctx context.Context, params CreateTransferParams) (TransferTxnRes, error) {
	var txnRes TransferTxnRes
	var err error

	if params.Amount.LessThanOrEqual(decimal.NewFromInt(0)) {
		return txnRes, errors.New("Transfer amount must be bigger than 0")
	}

	err = store.execTxn(ctx, func(queries *Queries) error {
		txnRes.Transfer, err = queries.CreateTransfer(ctx, params)
		if err != nil {
			return err
		}

		fromEntryParams := CreateEntryParams{
			AccountID: params.FromAccountID,
			Amount:    params.Amount.Neg(),
		}
		txnRes.FromEntry, err = queries.CreateEntry(ctx, fromEntryParams)
		if err != nil {
			return err
		}

		toEntryParams := CreateEntryParams{
			AccountID: params.ToAccountID,
			Amount:    params.Amount,
		}
		txnRes.ToEntry, err = queries.CreateEntry(ctx, toEntryParams)
		if err != nil {
			return err
		}

		// Update the balance of the two accounts in order (smallest ID first)
		// to avoid a DB deadlock
		if params.FromAccountID < params.ToAccountID {
			txnRes.FromAccount, err = queries.AddAccountBalance(
				ctx,
				AddAccountBalanceParams{
					ID:     params.FromAccountID,
					Amount: params.Amount.Neg(),
				},
			)

			if err != nil {
				return err
			}

			txnRes.ToAccount, err = queries.AddAccountBalance(
				ctx,
				AddAccountBalanceParams{
					ID:     params.ToAccountID,
					Amount: params.Amount,
				},
			)

			if err != nil {
				return err
			}
		} else {
			txnRes.ToAccount, err = queries.AddAccountBalance(
				ctx,
				AddAccountBalanceParams{
					ID:     params.ToAccountID,
					Amount: params.Amount,
				},
			)

			if err != nil {
				return err
			}

			txnRes.FromAccount, err = queries.AddAccountBalance(
				ctx,
				AddAccountBalanceParams{
					ID:     params.FromAccountID,
					Amount: params.Amount.Neg(),
				},
			)

			if err != nil {
				return err
			}
		}

		return nil
	})

	return txnRes, err
}
