package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/Andrew-2609/simple-bank/util"
	"github.com/stretchr/testify/require"
)

func createRandomAccount(t *testing.T) (account Account) {
	user := createRandomUser(t)

	arg := CreateAccountParams{
		Owner:    user.Username,
		Balance:  util.RandomAmount(),
		Currency: util.RandomCurrency(),
	}

	account, err := testQueries.CreateAccount(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, account)

	require.NotZero(t, account.ID)
	require.Equal(t, arg.Owner, account.Owner)
	require.Equal(t, arg.Balance, account.Balance)
	require.Equal(t, arg.Currency, account.Currency)
	require.NotZero(t, account.CreatedAt)

	return
}

func TestCreateAccount(t *testing.T) {
	createRandomAccount(t)
}

func TestGetAccount(t *testing.T) {
	account := createRandomAccount(t)

	foundAccount, err := testQueries.GetAccount(context.Background(), account.ID)

	require.NoError(t, err)
	require.NotEmpty(t, foundAccount)

	require.Exactly(t, account, foundAccount)
}

func TestGetAccountForUpdate(t *testing.T) {
	account := createRandomAccount(t)

	foundAccount, err := testQueries.GetAccountForUpdate(context.Background(), account.ID)

	require.NoError(t, err)
	require.NotEmpty(t, foundAccount)

	require.Exactly(t, account, foundAccount)
}

func TestListAccountsByOwner(t *testing.T) {
	var lastAccount Account

	for i := 0; i < 10; i++ {
		lastAccount = createRandomAccount(t)
	}

	arg := ListAccountsByOwnerParams{
		Owner:  lastAccount.Owner,
		Limit:  5,
		Offset: 0,
	}

	foundAccounts, err := testQueries.ListAccountsByOwner(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, foundAccounts)

	for _, account := range foundAccounts {
		require.NotEmpty(t, account)
		require.Equal(t, lastAccount.Owner, account.Owner)
	}
}

func TestUpdateAccount(t *testing.T) {
	originalAccount := createRandomAccount(t)

	var newBalance int64 = util.RandomAmount()

	arg := UpdateAccountParams{
		ID:      originalAccount.ID,
		Balance: newBalance,
	}

	updatedAccount, err := testQueries.UpdateAccount(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, updatedAccount)

	require.Equal(t, originalAccount.ID, updatedAccount.ID)
	require.Equal(t, originalAccount.Owner, updatedAccount.Owner)
	require.Equal(t, originalAccount.Currency, updatedAccount.Currency)
	require.Equal(t, originalAccount.CreatedAt, updatedAccount.CreatedAt)

	require.NotEqual(t, originalAccount.Balance, updatedAccount.Balance)
	require.Equal(t, newBalance, updatedAccount.Balance)
}

func TestDeleteAccount(t *testing.T) {
	account := createRandomAccount(t)

	err := testQueries.DeleteAccount(context.Background(), account.ID)
	require.NoError(t, err)

	foundAccount, err := testQueries.GetAccount(context.Background(), account.ID)

	require.EqualError(t, err, sql.ErrNoRows.Error())
	require.Empty(t, foundAccount)
}
