package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Andrew-2609/simple-bank/util"
	"github.com/stretchr/testify/require"
)

func createRandomEntry(t *testing.T) (entry Entry) {
	account := createRandomAccount(t)
	amount := int64(10)

	arg := CreateEntryParams{
		AccountID: account.ID,
		Amount:    amount,
	}

	entry, err := testQueries.CreateEntry(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, entry)

	require.NotZero(t, entry.ID)
	require.Equal(t, account.ID, entry.AccountID)
	require.Equal(t, amount, entry.Amount)
	require.NotZero(t, entry.CreatedAt)

	return
}

func TestCreateEntry(t *testing.T) {
	createRandomEntry(t)
}

func TestDeleteEntry(t *testing.T) {
	entry := createRandomEntry(t)

	err := testQueries.DeleteEntry(context.Background(), entry.ID)
	require.NoError(t, err)

	foundEntry, err := testQueries.GetEntry(context.Background(), entry.ID)

	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, foundEntry)
}

func TestGetEntry(t *testing.T) {
	entry := createRandomEntry(t)

	foundEntry, err := testQueries.GetEntry(context.Background(), entry.ID)
	require.NoError(t, err)

	require.Exactly(t, entry, foundEntry)
}

func TestListEntries(t *testing.T) {
	for i := 0; i < 10; i++ {
		createRandomEntry(t)
	}

	arg := ListEntriesParams{Limit: 5, Offset: 5}

	foundEntries, err := testQueries.ListEntries(context.Background(), arg)
	require.NoError(t, err)

	require.Len(t, foundEntries, 5)

	for _, entry := range foundEntries {
		require.NotEmpty(t, entry)
	}
}

func TestUpdateEntry(t *testing.T) {
	entry := createRandomEntry(t)

	var newAmount int64 = util.RandomAmount()

	arg := UpdateEntryParams{
		ID:     entry.ID,
		Amount: newAmount,
	}

	updatedEntry, err := testQueries.UpdateEntry(context.Background(), arg)
	require.NoError(t, err)

	require.NotEmpty(t, updatedEntry)

	require.Equal(t, entry.ID, updatedEntry.ID)
	require.Equal(t, entry.AccountID, updatedEntry.AccountID)
	require.Equal(t, entry.CreatedAt, updatedEntry.CreatedAt)

	require.NotEqual(t, entry.Amount, updatedEntry.Amount)
	require.Equal(t, newAmount, updatedEntry.Amount)
}
