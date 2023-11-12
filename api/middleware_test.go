package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Andrew-2609/simple-bank/token"
	"github.com/Andrew-2609/simple-bank/util"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func addAuthorization(t *testing.T, request *http.Request, tokenMaker token.Maker, authorizationType string, username string, duration time.Duration) {
	token, err := tokenMaker.CreateToken(username, duration)
	require.NoError(t, err)

	authorizationHeader := fmt.Sprintf("%s %s", authorizationType, token)

	request.Header.Set(authorizationHeaderKey, authorizationHeader)
}

func TestAuthMiddleware(t *testing.T) {
	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder httptest.ResponseRecorder)
	}{{
		name: "OK",
		setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			addAuthorization(t, request, tokenMaker, authorizationTypeBearer, "user", time.Minute)
		},
		checkResponse: func(t *testing.T, recorder httptest.ResponseRecorder) {
			require.Equal(t, http.StatusOK, recorder.Code)
		},
	}, {
		name:      "No Authorization Provided",
		setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {},
		checkResponse: func(t *testing.T, recorder httptest.ResponseRecorder) {
			require.Equal(t, http.StatusUnauthorized, recorder.Code)
			require.Exactly(t, map[string]interface{}{"error": "authorization header was not provided"}, UnmarshallAny(t, recorder.Body))
		},
	}, {
		name: "Invalid Authorization Header Format",
		setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			addAuthorization(t, request, tokenMaker, "", "user", time.Minute)
		},
		checkResponse: func(t *testing.T, recorder httptest.ResponseRecorder) {
			require.Equal(t, http.StatusUnauthorized, recorder.Code)
			require.Exactly(t, map[string]interface{}{"error": "invalid authorization header format"}, UnmarshallAny(t, recorder.Body))
		},
	}, {
		name: "Invalid Authorization Type",
		setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			addAuthorization(t, request, tokenMaker, "oauth", "user", time.Minute)
		},
		checkResponse: func(t *testing.T, recorder httptest.ResponseRecorder) {
			require.Equal(t, http.StatusUnauthorized, recorder.Code)
			require.Exactly(t, map[string]interface{}{"error": "invalid authorization type: oauth"}, UnmarshallAny(t, recorder.Body))
		},
	}, {
		name: "Invalid Token",
		setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			tokenMaker, err := token.NewPasetoMaker(util.RandomString(32))
			require.NoError(t, err)
			addAuthorization(t, request, tokenMaker, authorizationTypeBearer, "user", time.Minute)
		},
		checkResponse: func(t *testing.T, recorder httptest.ResponseRecorder) {
			require.Equal(t, http.StatusUnauthorized, recorder.Code)
			require.Exactly(t, map[string]interface{}{"error": "invalid token"}, UnmarshallAny(t, recorder.Body))
		},
	}, {
		name: "Expired Token",
		setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
			addAuthorization(t, request, tokenMaker, authorizationTypeBearer, "user", -time.Minute)
		},
		checkResponse: func(t *testing.T, recorder httptest.ResponseRecorder) {
			require.Equal(t, http.StatusUnauthorized, recorder.Code)
			require.Exactly(t, map[string]interface{}{"error": "token has expired"}, UnmarshallAny(t, recorder.Body))
		},
	}}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			server := newTestServer(t, nil)

			authPath := "/auth"

			server.router.GET(
				authPath,
				authMiddleware(server.tokenMaker),
				func(ctx *gin.Context) {
					ctx.JSON(http.StatusOK, gin.H{})
				},
			)

			recorder := httptest.NewRecorder()

			request, err := http.NewRequest("GET", authPath, nil)
			require.NoError(t, err)

			testCase.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)

			testCase.checkResponse(t, *recorder)
		})
	}
}
