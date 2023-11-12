package token

import (
	"fmt"
	"testing"
	"time"

	"github.com/Andrew-2609/simple-bank/util"
	"github.com/stretchr/testify/require"
)

func createPasetoToken(t *testing.T, maker Maker) (pasetoToken string, protoPayload *Payload) {
	username := util.RandomOwner()
	duration := time.Minute
	issuedAt := time.Now()
	expiredAt := issuedAt.Add(duration)

	protoPayload = &Payload{
		Username:  username,
		IssuedAt:  issuedAt,
		ExpiredAt: expiredAt,
	}

	pasetoToken, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, pasetoToken)

	return
}

func TestPasetoMaker(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	pasetoToken, protoPayload := createPasetoToken(t, maker)

	payload, err := maker.VerifyToken(pasetoToken)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	require.NotZero(t, payload.ID)
	require.Equal(t, protoPayload.Username, payload.Username)
	require.WithinDuration(t, protoPayload.IssuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, protoPayload.ExpiredAt, payload.ExpiredAt, time.Second)
}

func TestExpiredPasetoToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	pasetoToken, err := maker.CreateToken(util.RandomOwner(), -time.Second)
	require.NoError(t, err)

	payload, err := maker.VerifyToken(pasetoToken)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Nil(t, payload)
}

func TestPasetoMakerCreateTooShortToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(31))
	require.Error(t, err)
	require.EqualError(t, err, fmt.Errorf("invalid key size: must have exactly 32 characters.").Error())
	require.Nil(t, maker)
}

func TestInvalidPasetoToken(t *testing.T) {
	maker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	anotherMaker, err := NewPasetoMaker(util.RandomString(32))
	require.NoError(t, err)

	token, err := anotherMaker.CreateToken(util.RandomOwner(), time.Minute)
	require.NoError(t, err)

	payload, err := maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrInvalidToken.Error())
	require.Nil(t, payload)
}
