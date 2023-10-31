package db

import (
	"context"
	"testing"

	"github.com/Andrew-2609/simple-bank/util"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) (user User) {
	hashedPassword, err := util.HashPassword(util.RandomString(8))
	require.NoError(t, err)

	arg := CreateUserParams{
		Username: util.RandomOwner(),
		Password: hashedPassword,
		Name:     util.RandomOwner(),
		LastName: util.RandomOwner(),
		Email:    util.RandomEmail(),
	}

	user, err = testQueries.CreateUser(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.Password, user.Password)
	require.Equal(t, arg.Name, user.Name)
	require.Equal(t, arg.LastName, user.LastName)
	require.Equal(t, arg.Email, user.Email)
	require.True(t, user.PasswordChangedAt.IsZero())
	require.NotZero(t, user.CreatedAt)

	return
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestGetUser(t *testing.T) {
	user := createRandomUser(t)

	foundUser, err := testQueries.GetUser(context.Background(), user.Username)

	require.NoError(t, err)
	require.NotEmpty(t, foundUser)

	require.Exactly(t, user, foundUser)
}
