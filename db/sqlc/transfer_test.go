package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Andrew-2609/simple-bank/util"
	"github.com/stretchr/testify/require"
)

func createRandomTransfer(t *testing.T) (transfer Transfer) {
	fromAccount := createRandomAccount(t)
	toAccount := createRandomAccount(t)

	amount := int64(10)

	arg := CreateTransferParams{
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		Amount:        amount,
	}

	transfer, err := testQueries.CreateTransfer(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, transfer)

	require.NotZero(t, transfer.ID)
	require.Equal(t, fromAccount.ID, transfer.FromAccountID)
	require.Equal(t, toAccount.ID, transfer.ToAccountID)
	require.Equal(t, amount, transfer.Amount)
	require.NotZero(t, transfer.CreatedAt)

	return
}

func TestCreateTransfer(t *testing.T) {
	createRandomTransfer(t)
}

func TestDeleteTransfer(t *testing.T) {
	transfer := createRandomTransfer(t)

	err := testQueries.DeleteTransfer(context.Background(), transfer.ID)
	require.NoError(t, err)

	foundTransfer, err := testQueries.GetTransfer(context.Background(), transfer.ID)

	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, foundTransfer)
}

func TestGetTransfer(t *testing.T) {
	transfer := createRandomTransfer(t)

	foundTransfer, err := testQueries.GetTransfer(context.Background(), transfer.ID)
	require.NoError(t, err)

	require.Exactly(t, transfer, foundTransfer)
}

func TestListTransfers(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomTransfer(t)
	}

	arg := ListTransfersParams{Limit: 5, Offset: 5}

	transfers, err := testQueries.ListTransfers(context.Background(), arg)
	require.NoError(t, err)

	require.Len(t, transfers, 5)

	for _, transfer := range transfers {
		require.NotEmpty(t, transfer)
	}
}

func TestUpdateTransfer(t *testing.T) {
	transfer := createRandomTransfer(t)

	var newAmount = util.RandomAmount()

	arg := UpdateTransferParams{
		ID:     transfer.ID,
		Amount: newAmount,
	}

	updatedTransfer, err := testQueries.UpdateTransfer(context.Background(), arg)
	require.NoError(t, err)

	require.Equal(t, transfer.ID, updatedTransfer.ID)
	require.Equal(t, transfer.FromAccountID, updatedTransfer.FromAccountID)
	require.Equal(t, transfer.ToAccountID, updatedTransfer.ToAccountID)
	require.Equal(t, transfer.CreatedAt, updatedTransfer.CreatedAt)

	require.NotEqual(t, transfer.Amount, updatedTransfer.Amount)
	require.Equal(t, newAmount, updatedTransfer.Amount)
}
