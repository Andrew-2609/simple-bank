package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestCorrectPassword(t *testing.T) {
	password := RandomString(8)

	hashedPassword, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword)

	err = CheckPassword(hashedPassword, password)
	require.NoError(t, err)
}

func TestWrongPassword(t *testing.T) {
	password := RandomString(8)

	hashedPassword, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, hashedPassword)

	wrongPassword := RandomString(8)

	err = CheckPassword(hashedPassword, wrongPassword)
	require.EqualError(t, err, bcrypt.ErrMismatchedHashAndPassword.Error())
}

func TestDifferentHashPasswordOutputs(t *testing.T) {
	password := RandomString(8)

	firstHashedPassword, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, firstHashedPassword)

	err = CheckPassword(firstHashedPassword, password)
	require.NoError(t, err)

	secondHashedPassword, err := HashPassword(password)
	require.NoError(t, err)
	require.NotEmpty(t, secondHashedPassword)

	err = CheckPassword(secondHashedPassword, password)
	require.NoError(t, err)

	require.NotEqual(t, firstHashedPassword, secondHashedPassword)
}
