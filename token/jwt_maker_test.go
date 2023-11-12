package token

import (
	"fmt"
	"testing"
	"time"

	"github.com/Andrew-2609/simple-bank/util"
	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

func createJWTToken(t *testing.T, maker Maker) (jwtToken string, protoPayload *Payload) {
	username := util.RandomOwner()
	duration := time.Minute
	issuedAt := time.Now()
	expiredAt := issuedAt.Add(duration)

	protoPayload = &Payload{
		Username:  username,
		IssuedAt:  issuedAt,
		ExpiredAt: expiredAt,
	}

	jwtToken, err := maker.CreateToken(username, duration)
	require.NoError(t, err)
	require.NotEmpty(t, jwtToken)

	return
}

func TestJWTMakerCreateTooShortToken(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(31))
	require.Error(t, err)
	require.EqualError(t, err, fmt.Errorf("Invalid secret key size: must have at least 32 characters.").Error())
	require.Nil(t, maker)
}

func TestJWTMakerVerifyToken(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	jwtToken, protoPayload := createJWTToken(t, maker)

	payload, err := maker.VerifyToken(jwtToken)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	require.NotZero(t, payload.ID)
	require.Equal(t, protoPayload.Username, payload.Username)
	require.WithinDuration(t, protoPayload.IssuedAt, payload.IssuedAt, time.Second)
	require.WithinDuration(t, protoPayload.ExpiredAt, payload.ExpiredAt, time.Second)
}

func TestExpiredJWTToken(t *testing.T) {
	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	jwtToken, err := maker.CreateToken(util.RandomOwner(), -time.Second)
	require.NoError(t, err)

	payload, err := maker.VerifyToken(jwtToken)
	require.Error(t, err)
	require.EqualError(t, err, ErrExpiredToken.Error())
	require.Nil(t, payload)
}

func TestInvalidJWTTokenAlgNone(t *testing.T) {
	payload, err := NewPayload(util.RandomOwner(), time.Minute)
	require.NoError(t, err)

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodNone, payload)

	token, err := jwtToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	maker, err := NewJWTMaker(util.RandomString(32))
	require.NoError(t, err)

	payload, err = maker.VerifyToken(token)
	require.Error(t, err)
	require.EqualError(t, err, ErrInvalidToken.Error())
	require.Nil(t, payload)
}
