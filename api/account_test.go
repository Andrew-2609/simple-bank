package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	mockdb "github.com/Andrew-2609/simple-bank/db/mock"
	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	"github.com/Andrew-2609/simple-bank/token"
	"github.com/Andrew-2609/simple-bank/util"
)

func createRandomAccount(owner string) db.Account {
	return db.Account{
		ID:       util.RandomInt(1, 1000),
		Owner:    owner,
		Balance:  util.RandomAmount(),
		Currency: util.RandomCurrency(),
	}
}

func unmarshallAccount(t *testing.T, responseBody *bytes.Buffer) db.Account {
	responseAccount, err := util.UnmarshallJsonBody[db.Account](responseBody)
	require.NoError(t, err)
	return responseAccount
}

func TestGetAccountAPI(t *testing.T) {
	user, _ := createRandomUser()

	account := createRandomAccount(user.Username)

	testCases := []struct {
		name          string
		accountId     int64
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountId: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				require.Exactly(t, account, unmarshallAccount(t, recorder.Body))
			},
		},
		{
			name:      "Unauthorized",
			accountId: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, "unauthorized", time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "account doesn't belong to the authenticated user"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name:       "No Authorization",
			accountId:  account.ID,
			setupAuth:  func(t *testing.T, request *http.Request, tokenMaker token.Maker) {},
			buildStubs: func(store *mockdb.MockStore) { store.EXPECT().GetAccount(gomock.Any(), 0).Times(0) },
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "authorization header was not provided"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name:      "Bad Request",
			accountId: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), 0).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'getAccountRequest.ID' Error:Field validation for 'ID' failed on the 'required' tag"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name:      "Not Found",
			accountId: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrNoRows)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrNoRows.Error()}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name:      "Internal Server Error",
			accountId: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, sql.ErrConnDone)
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

			url := fmt.Sprintf("/accounts/%d", testCase.accountId)

			request, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			// setup authorization middleware
			testCase.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)

			// check response
			testCase.checkResponse(t, recorder)
		})
	}
}

func TestCreateAccountAPI(t *testing.T) {
	user, _ := createRandomUser()

	expectedAccount := createRandomAccount(user.Username)

	validArg := db.CreateAccountParams{
		Owner:    user.Username,
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
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "Created",
			arg:  validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
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
			name:       "No Authorization",
			arg:        validArg,
			setupAuth:  func(t *testing.T, request *http.Request, tokenMaker token.Maker) {},
			buildStubs: func(store *mockdb.MockStore) { store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(0) },
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "authorization header was not provided"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Bad Request",
			arg:  db.CreateAccountParams{},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().CreateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'createAccountRequest.Currency' Error:Field validation for 'Currency' failed on the 'required' tag"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Internal Server Error",
			arg:  validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(expectedArg)).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)
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

			url := fmt.Sprintf("/accounts")

			var buf bytes.Buffer

			err := json.NewEncoder(&buf).Encode(testCase.arg)
			require.NoError(t, err)

			request, err := http.NewRequest("POST", url, &buf)
			require.NoError(t, err)

			// setup authorization middleware
			testCase.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)

			// check response
			testCase.checkResponse(t, recorder)
		})
	}
}

func TestListAccountsAPI(t *testing.T) {
	user, _ := createRandomUser()

	accounts := []db.Account{createRandomAccount(user.Username), createRandomAccount(user.Username)}

	expectedSpecifiedArg := db.ListAccountsByOwnerParams{
		Owner:  user.Username,
		Limit:  3,
		Offset: 0,
	}

	expectedDefaultArg := db.ListAccountsByOwnerParams{
		Owner:  user.Username,
		Limit:  40,
		Offset: 0,
	}

	testCases := []struct {
		name          string
		page          int32
		quantity      int32
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:     "OK With Specified Parameters Values",
			page:     1,
			quantity: 3,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccountsByOwner(gomock.Any(), gomock.Eq(expectedSpecifiedArg)).
					Times(1).
					Return(accounts, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				fmt.Printf("Array format: %v", recorder.Body)
			},
		},
		{
			name:     "OK With Default Parameters Values",
			page:     1,
			quantity: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccountsByOwner(gomock.Any(), gomock.Eq(expectedDefaultArg)).
					Times(1).
					Return(accounts, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				fmt.Printf("Array format: %v", recorder.Body)
			},
		},
		{
			name:       "No Authorization",
			page:       1,
			quantity:   3,
			setupAuth:  func(t *testing.T, request *http.Request, tokenMaker token.Maker) {},
			buildStubs: func(store *mockdb.MockStore) { store.EXPECT().ListAccountsByOwner(gomock.Any(), gomock.Any()).Times(0) },
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "authorization header was not provided"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name:     "Bad Request",
			page:     -1,
			quantity: 3,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().ListAccountsByOwner(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'listAccountsRequest.Page' Error:Field validation for 'Page' failed on the 'min' tag"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name:     "Internal Server Error",
			page:     1,
			quantity: 3,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccountsByOwner(gomock.Any(), gomock.Eq(expectedSpecifiedArg)).
					Times(1).
					Return([]db.Account{}, sql.ErrConnDone)
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

			url := fmt.Sprintf("/accounts?page=%d&quantity=%d", testCase.page, testCase.quantity)

			request, err := http.NewRequest("GET", url, nil)
			require.NoError(t, err)

			// setup authorization middleware
			testCase.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)

			// check response
			testCase.checkResponse(t, recorder)
		})
	}
}

func TestUpdateAccountAPI(t *testing.T) {
	user, _ := createRandomUser()

	originalAccount := createRandomAccount(user.Username)

	validArg := db.UpdateAccountParams{
		ID:      originalAccount.ID,
		Balance: 5000,
	}

	updatedAccount := db.Account{
		ID:        originalAccount.ID,
		Owner:     originalAccount.Owner,
		Balance:   validArg.Balance,
		Currency:  originalAccount.Currency,
		CreatedAt: originalAccount.CreatedAt,
	}

	testCases := []struct {
		name          string
		arg           db.UpdateAccountParams
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			arg:  validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(validArg.ID)).
					Times(1).
					Return(originalAccount, nil)
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Eq(validArg)).
					Times(1).
					Return(updatedAccount, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				require.Exactly(t, updatedAccount, unmarshallAccount(t, recorder.Body))
			},
		},
		{
			name: "Unauthorized",
			arg:  validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, "unauthorized", time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(validArg.ID)).
					Times(1).
					Return(originalAccount, nil)
				store.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "account doesn't belong to the authenticated user"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name:      "No Authorization",
			arg:       validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "authorization header was not provided"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Bad Request with Wrong Params",
			arg:  db.UpdateAccountParams{ID: -1, Balance: 3500},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'ID' Error:Field validation for 'ID' failed on the 'min' tag"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Bad Request with Wrong Body",
			arg:  db.UpdateAccountParams{ID: originalAccount.ID, Balance: -3000},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'Balance' Error:Field validation for 'Balance' failed on the 'min' tag"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Not Found",
			arg:  validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(validArg.ID)).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)
				store.EXPECT().UpdateAccount(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrNoRows.Error()}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Internal Server Error",
			arg:  validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(validArg.ID)).
					Times(1).
					Return(originalAccount, nil)
				store.EXPECT().
					UpdateAccount(gomock.Any(), gomock.Eq(validArg)).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)
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

			url := fmt.Sprintf("/accounts/%d", testCase.arg.ID)

			var buf bytes.Buffer

			err := json.NewEncoder(&buf).Encode(testCase.arg)
			require.NoError(t, err)

			request, err := http.NewRequest("PUT", url, &buf)
			require.NoError(t, err)

			// setup authorization middleware
			testCase.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)

			// check resposne
			testCase.checkResponse(t, recorder)
		})
	}
}

func TestDeleteAccountAPI(t *testing.T) {
	user, _ := createRandomUser()

	account := createRandomAccount(user.Username)

	testCases := []struct {
		name          string
		accountId     int64
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "No Content",
			accountId: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, recorder.Code)
			},
		},
		{
			name:      "Bad Request",
			accountId: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteAccount(gomock.Any(), 0).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'deleteAccountRequest.ID' Error:Field validation for 'ID' failed on the 'required' tag"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name:      "Internal Server Error",
			accountId: account.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().DeleteAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(sql.ErrConnDone)
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

			url := fmt.Sprintf("/accounts/%d", testCase.accountId)

			request, err := http.NewRequest("DELETE", url, nil)
			require.NoError(t, err)

			// setup authorization middleware
			testCase.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)

			// check response
			testCase.checkResponse(t, recorder)
		})
	}
}
