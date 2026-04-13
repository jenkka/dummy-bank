package db

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestTransferTxn(t *testing.T) {
	store := NewStore(testDB)
	ogFromAccount := createRandomAccount(t)
	ogToAccount := createRandomAccount(t)

	nTransfers := 5
	transferAmount := decimal.NewFromFloat(10.0)
	transferParams := CreateTransferParams{
		FromAccountID: ogFromAccount.ID,
		ToAccountID:   ogToAccount.ID,
		Amount:        transferAmount,
	}

	errors := make(chan error)
	results := make(chan TransferTxnRes)

	for i := 0; i < nTransfers; i++ {
		go func() {
			res, err := store.TransferTxn(context.Background(), transferParams)
			errors <- err
			results <- res
		}()
	}

	for i := 0; i < nTransfers; i++ {
		require.NoError(t, <-errors)

		res := <-results

		resTransfer := res.Transfer
		require.NotEmpty(t, resTransfer.ID)
		require.NotEmpty(t, resTransfer.CreatedAt)
		requireDecimalEqual(t, transferParams.Amount, resTransfer.Amount)
		require.Equal(t, transferParams.FromAccountID, resTransfer.FromAccountID)
		require.Equal(t, transferParams.ToAccountID, resTransfer.ToAccountID)

		_, err := testQueries.GetTransfer(context.Background(), resTransfer.ID)
		require.NoError(t, err)

		resFromEntry := res.FromEntry
		require.NotEmpty(t, resFromEntry.ID)
		require.NotEmpty(t, resFromEntry.CreatedAt)
		requireDecimalEqual(t, transferParams.Amount.Neg(), resFromEntry.Amount)
		require.Equal(t, transferParams.FromAccountID, resFromEntry.AccountID)

		_, err = testQueries.GetEntry(context.Background(), resFromEntry.ID)
		require.NoError(t, err)

		resToEntry := res.ToEntry
		require.NotEmpty(t, resToEntry.ID)
		require.NotEmpty(t, resToEntry.CreatedAt)
		requireDecimalEqual(t, transferParams.Amount, resToEntry.Amount)
		require.Equal(t, transferParams.ToAccountID, resToEntry.AccountID)

		_, err = testQueries.GetEntry(context.Background(), resToEntry.ID)
		require.NoError(t, err)

		resFromAccount := res.FromAccount
		require.NotEmpty(t, resFromAccount)
		require.Equal(t, ogFromAccount.ID, resFromAccount.ID)

		resToAccount := res.ToAccount
		require.NotEmpty(t, resToAccount)
		require.Equal(t, ogToAccount.ID, resToAccount.ID)

		fromAccountDiff := ogFromAccount.Balance.Sub(resFromAccount.Balance)
		toAccountDiff := resToAccount.Balance.Sub(ogToAccount.Balance)
		requireDecimalEqual(t, fromAccountDiff, toAccountDiff)
		require.True(t, fromAccountDiff.GreaterThan(decimal.NewFromInt(0)))
	}

	updatedFromAccount, err := testQueries.GetAccount(context.Background(), ogFromAccount.ID)
	require.NoError(t, err)
	require.NotEmpty(t, updatedFromAccount)

	updatedToAccount, err := testQueries.GetAccount(context.Background(), ogToAccount.ID)
	require.NoError(t, err)
	require.NotEmpty(t, updatedToAccount)

	expectedFromAccountBalance := ogFromAccount.Balance.Sub(transferAmount.Mul(decimal.NewFromInt(int64(nTransfers))))
	requireDecimalEqual(t, expectedFromAccountBalance, updatedFromAccount.Balance)

	expectedToAccountBalance := ogToAccount.Balance.Add(transferAmount.Mul(decimal.NewFromInt(int64(nTransfers))))
	requireDecimalEqual(t, expectedToAccountBalance, updatedToAccount.Balance)
}

func TestTransferTxnDeadlock(t *testing.T) {
	store := NewStore(testDB)
	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	nTransfers := 10
	transferAmount := decimal.NewFromFloat(10.0)

	errors := make(chan error)

	for i := 0; i < nTransfers; i++ {
		fromAccountID := account1.ID
		toAccountID := account2.ID

		if i%2 != 0 {
			fromAccountID = account2.ID
			toAccountID = account1.ID
		}

		go func() {
			transferParams := CreateTransferParams{
				FromAccountID: fromAccountID,
				ToAccountID:   toAccountID,
				Amount:        transferAmount,
			}
			_, err := store.TransferTxn(context.Background(), transferParams)
			errors <- err
		}()
	}

	for i := 0; i < nTransfers; i++ {
		require.NoError(t, <-errors)
	}

	updatedAccount1, err := testQueries.GetAccount(context.Background(), account1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAccount1)

	updatedAccount2, err := testQueries.GetAccount(context.Background(), account2.ID)
	require.NoError(t, err)
	require.NotEmpty(t, updatedAccount2)

	requireDecimalEqual(t, account1.Balance, updatedAccount1.Balance)
	requireDecimalEqual(t, account2.Balance, updatedAccount2.Balance)
}
