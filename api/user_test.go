package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	mockdb "github.com/Andrew-2609/simple-bank/db/mock"
	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	"github.com/Andrew-2609/simple-bank/util"
	"github.com/golang/mock/gomock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

type eqCreateUserParamsMatcher struct {
	arg         db.CreateUserParams
	rawPassword string
}

func (eq eqCreateUserParamsMatcher) Matches(x interface{}) bool {
	arg, ok := x.(db.CreateUserParams)

	if !ok {
		return false
	}

	if err := util.CheckPassword(arg.HashedPassword, eq.rawPassword); err != nil {
		return false
	}

	eq.arg.HashedPassword = arg.HashedPassword

	return reflect.DeepEqual(eq.arg, arg)
}

func (eq eqCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("%v (%T)\nDon't mind the hashed password. What matters is the unhashed value, that must be \"%s\"", eq.arg, eq.arg, eq.rawPassword)
}

func EqCreateUserParams(arg db.CreateUserParams, rawPassword string) gomock.Matcher {
	return eqCreateUserParamsMatcher{arg, rawPassword}
}

func createRandomUser() db.User {
	hashedPassword, _ := util.HashPassword(util.RandomString(8))

	return db.User{
		Username:       util.RandomOwner(),
		HashedPassword: hashedPassword,
		Name:           util.RandomString(5),
		LastName:       util.RandomString(8),
		Email:          util.RandomEmail(),
	}
}

func unmarshallUser(t *testing.T, responseBody *bytes.Buffer) db.User {
	responseUser, err := util.UnmarshallJsonBody[db.User](responseBody)
	require.NoError(t, err)
	return responseUser
}

func TestCreateUserAPI(t *testing.T) {
	expectedUser := createRandomUser()

	validBody := CreateUserRequest{
		Username: util.RandomOwner(),
		Password: util.RandomString(8),
		Name:     util.RandomString(5),
		LastName: util.RandomString(8),
		Email:    util.RandomEmail(),
	}

	hashedPassword, _ := util.HashPassword(validBody.Password)

	testCases := []struct {
		name          string
		body          CreateUserRequest
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "Created",
			body: validBody,
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateUserParams{
					Username:       validBody.Username,
					HashedPassword: hashedPassword,
					Name:           validBody.Name,
					LastName:       validBody.LastName,
					Email:          validBody.Email,
				}

				store.EXPECT().
					CreateUser(gomock.Any(), EqCreateUserParams(arg, validBody.Password)).
					Times(1).
					Return(expectedUser, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				require.Exactly(t, db.User{
					Username:          expectedUser.Username,
					Name:              expectedUser.Name,
					LastName:          expectedUser.LastName,
					Email:             expectedUser.Email,
					PasswordChangedAt: expectedUser.PasswordChangedAt,
					CreatedAt:         expectedUser.CreatedAt,
				}, unmarshallUser(t, recorder.Body))
			},
		},
		{
			name: "Bad Request",
			body: CreateUserRequest{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'CreateUserRequest.Username' Error:Field validation for 'Username' failed on the 'required' tag\nKey: 'CreateUserRequest.Password' Error:Field validation for 'Password' failed on the 'required' tag\nKey: 'CreateUserRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag\nKey: 'CreateUserRequest.LastName' Error:Field validation for 'LastName' failed on the 'required' tag\nKey: 'CreateUserRequest.Email' Error:Field validation for 'Email' failed on the 'required' tag"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "HashPassword Internal Error",
			body: CreateUserRequest{
				Username: validBody.Username,
				Password: util.RandomString(73),
				Name:     validBody.Name,
				LastName: validBody.LastName,
				Email:    validBody.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Failed to hash password: bcrypt: password length exceeds 72 bytes"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Unique Violation",
			body: validBody,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, &pq.Error{
						Code:    pq.ErrorCode("23505"),
						Message: "duplicate key value violates unique constraint \"users_pkey\"",
					})
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "pq: duplicate key value violates unique constraint \"users_pkey\""}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Internal Server Error",
			body: validBody,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrConnDone.Error()}, UnmarshallAny(t, recorder.Body))
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// build stubs
			store := mockdb.NewMockStore(ctrl)
			testCase.buildStubs(store)

			// start test server and send request
			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/users")

			var buf bytes.Buffer

			err := json.NewEncoder(&buf).Encode(testCase.body)
			require.NoError(t, err)

			request, err := http.NewRequest("POST", url, &buf)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			// check response
			testCase.checkResponse(t, recorder)
		})
	}
}
