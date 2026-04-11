package db

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTransferTxn(t *testing.T) {
	store := NewStore(testDB)
	ogFromAccount := createRandomAccount(t)
	ogToAccount := createRandomAccount(t)

	ogFromAccountBalance, err := strconv.ParseFloat(ogFromAccount.Balance, 64)
	require.NoError(t, err)

	ogToAccountBalance, err := strconv.ParseFloat(ogToAccount.Balance, 64)
	require.NoError(t, err)

	nTransfers := 5
	transferAmount := 10.0
	transferParams := CreateTransferParams{
		FromAccountID: ogFromAccount.ID,
		ToAccountID:   ogToAccount.ID,
		Amount:        fmt.Sprintf("%v", transferAmount),
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
		require.Equal(t, transferParams.Amount, resTransfer.Amount)
		require.Equal(t, transferParams.FromAccountID, resTransfer.FromAccountID)
		require.Equal(t, transferParams.ToAccountID, resTransfer.ToAccountID)

		_, err = testQueries.GetTransfer(context.Background(), resTransfer.ID)
		require.NoError(t, err)

		resFromEntry := res.FromEntry
		require.NotEmpty(t, resFromEntry.ID)
		require.NotEmpty(t, resFromEntry.CreatedAt)
		require.Equal(t, fmt.Sprintf("-%s", transferParams.Amount), resFromEntry.Amount)
		require.Equal(t, transferParams.FromAccountID, resFromEntry.AccountID)

		_, err = testQueries.GetEntry(context.Background(), resFromEntry.ID)
		require.NoError(t, err)

		resToEntry := res.ToEntry
		require.NotEmpty(t, resToEntry.ID)
		require.NotEmpty(t, resToEntry.CreatedAt)
		require.Equal(t, transferParams.Amount, resToEntry.Amount)
		require.Equal(t, transferParams.ToAccountID, resToEntry.AccountID)

		_, err = testQueries.GetEntry(context.Background(), resToEntry.ID)
		require.NoError(t, err)

		resFromAccount := res.FromAccount
		require.NotEmpty(t, resFromAccount)
		require.Equal(t, ogFromAccount.ID, resFromAccount.ID)

		resFromAccountBalance, err := strconv.ParseFloat(resFromAccount.Balance, 64)
		require.NoError(t, err)

		resToAccount := res.ToAccount
		require.NotEmpty(t, resToAccount)
		require.Equal(t, ogToAccount.ID, resToAccount.ID)

		resToAccountBalance, err := strconv.ParseFloat(resToAccount.Balance, 64)
		require.NoError(t, err)

		fromAccountDiff := ogFromAccountBalance - resFromAccountBalance
		toAccountDiff := resToAccountBalance - ogToAccountBalance
		require.Equal(t, fromAccountDiff, toAccountDiff)
		require.True(t, fromAccountDiff > 0)
	}

	updatedFromAccount, err := testQueries.GetAccount(context.Background(), ogFromAccount.ID)
	require.NoError(t, err)
	require.NotEmpty(t, updatedFromAccount)

	updatedToAccount, err := testQueries.GetAccount(context.Background(), ogToAccount.ID)
	require.NoError(t, err)
	require.NotEmpty(t, updatedToAccount)

	expectedFromAccountBalance := fmt.Sprintf("%v", ogFromAccountBalance-transferAmount*float64(nTransfers))
	require.Equal(t, expectedFromAccountBalance, updatedFromAccount.Balance)

	expectedToAccountBalance := fmt.Sprintf("%v", ogToAccountBalance+transferAmount*float64(nTransfers))
	require.Equal(t, expectedToAccountBalance, updatedToAccount.Balance)
}
