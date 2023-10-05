package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	mockdb "github.com/Andrew-2609/simple-bank/db/mock"
	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	"github.com/Andrew-2609/simple-bank/util"
)

func createRandomAccount() db.Account {
	return db.Account{
		ID:       util.RandomInt(1, 1000),
		Owner:    util.RandomOwner(),
		Balance:  util.RandomAmount(),
		Currency: util.RandomCurrency(),
	}
}

func unmarshallAccount(t *testing.T, responseBody *bytes.Buffer) db.Account {
	responseAccount, err := util.UnmarshallJsonBody[db.Account](responseBody)
	require.NoError(t, err)
	return responseAccount
}

func unmarshallAny(t *testing.T, responseBody *bytes.Buffer) any {
	unmarshalledObject, err := util.UnmarshallJsonBody[any](responseBody)
	require.NoError(t, err)
	return unmarshalledObject
}

func TestGetAccountAPI(t *testing.T) {
	account := createRandomAccount()

	testCases := []struct {
		name          string
		accountId     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountId: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				require.Exactly(t, account, unmarshallAccount(t, recorder.Body))
			},
		},
		{
			name:      "Bad Request",
			accountId: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), 0).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'getAccountRequest.ID' Error:Field validation for 'ID' failed on the 'required' tag"}, unmarshallAny(t, recorder.Body))
			},
		},
		{
			name:      "Not Found",
			accountId: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrNoRows.Error()}, unmarshallAny(t, recorder.Body))
			},
		},
		{
			name:      "Internal Server Error",
			accountId: account.ID,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrConnDone.Error()}, unmarshallAny(t, recorder.Body))
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
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts/%d", testCase.accountId)

			request, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			// check response
			testCase.checkResponse(t, recorder)
		})
	}
}

func TestCreateAccountAPI(t *testing.T) {
	expectedAccount := createRandomAccount()

	validArg := db.CreateAccountParams{
		Owner:    util.RandomOwner(),
		Currency: util.RandomCurrency(),
	}

	expectedArg := db.CreateAccountParams{
		Owner:    validArg.Owner,
		Balance:  0,
		Currency: validArg.Currency,
	}

	testCases := []struct {
		name          string
		arg           db.CreateAccountParams
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "Created",
			arg:  validArg,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(expectedArg)).
					Times(1).
					Return(expectedAccount, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				require.Exactly(t, expectedAccount, unmarshallAccount(t, recorder.Body))
			},
		},
		{
			name: "Bad Request",
			arg:  db.CreateAccountParams{},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'createAccountRequest.Owner' Error:Field validation for 'Owner' failed on the 'required' tag\nKey: 'createAccountRequest.Currency' Error:Field validation for 'Currency' failed on the 'required' tag"}, unmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Internal Server Error",
			arg:  validArg,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(expectedArg)).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrConnDone.Error()}, unmarshallAny(t, recorder.Body))
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
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts")

			var buf bytes.Buffer

			err := json.NewEncoder(&buf).Encode(testCase.arg)
			require.NoError(t, err)

			request, err := http.NewRequest("POST", url, &buf)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			// check response
			testCase.checkResponse(t, recorder)
		})
	}
}

func TestListAccountsAPI(t *testing.T) {
	accounts := []db.Account{createRandomAccount(), createRandomAccount()}

	expectedSpecifiedArg := db.ListAccountsParams{
		Limit:  3,
		Offset: 0,
	}

	expectedDefaultArg := db.ListAccountsParams{
		Limit:  40,
		Offset: 0,
	}

	testCases := []struct {
		name          string
		page          int32
		quantity      int32
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "OK With Specified Parameters Values",
			page:     1,
			quantity: 3,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(expectedSpecifiedArg)).
					Times(1).
					Return(accounts, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				fmt.Printf("Array format: %v", recorder.Body)
			},
		}, {
			name:     "OK With Default Parameters Values",
			page:     1,
			quantity: 0,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(expectedDefaultArg)).
					Times(1).
					Return(accounts, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				fmt.Printf("Array format: %v", recorder.Body)
			},
		}, {
			name:     "Bad Request",
			page:     -1,
			quantity: 3,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListAccounts(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'listAccountsRequest.Page' Error:Field validation for 'Page' failed on the 'min' tag"}, unmarshallAny(t, recorder.Body))
			},
		}, {
			name:     "Internal Server Error",
			page:     1,
			quantity: 3,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(expectedSpecifiedArg)).
					Times(1).
					Return([]db.Account{}, sql.ErrConnDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrConnDone.Error()}, unmarshallAny(t, recorder.Body))
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
			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/accounts?page=%d&quantity=%d", testCase.page, testCase.quantity)

			request, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			// check response
			testCase.checkResponse(t, recorder)
		})
	}
}
