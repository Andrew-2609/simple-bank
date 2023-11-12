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

	mockdb "github.com/Andrew-2609/simple-bank/db/mock"
	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	"github.com/Andrew-2609/simple-bank/token"
	"github.com/Andrew-2609/simple-bank/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func createRandomAccounts() [2]db.Account {
	fromUser, _ := createRandomUser()
	toUser, _ := createRandomUser()

	return [2]db.Account{
		{
			ID:       util.RandomInt(1, 1000),
			Owner:    fromUser.Username,
			Balance:  util.RandomAmount(),
			Currency: util.BRL,
		}, {
			ID:       util.RandomInt(1, 1000),
			Owner:    toUser.Username,
			Balance:  util.RandomAmount(),
			Currency: util.BRL,
		},
	}
}

func unmarshallTransfer(t *testing.T, responseBody *bytes.Buffer) db.TransferTxResult {
	responseTransfer, err := util.UnmarshallJsonBody[db.TransferTxResult](responseBody)
	require.NoError(t, err)
	return responseTransfer
}

func TestCreateTransferAPI(t *testing.T) {
	accounts := createRandomAccounts()
	var amount int64 = 5000

	validArg := CreateTransferRequest{
		FromAccountID: accounts[0].ID,
		ToAccountID:   accounts[1].ID,
		Amount:        amount,
		Currency:      "BRL",
	}

	expectedArg := db.TransferTxParams{
		FromAccountID: accounts[0].ID,
		ToAccountID:   accounts[1].ID,
		Amount:        amount,
	}

	expectedResult := db.TransferTxResult{
		Transfer: db.Transfer{
			ID:            1,
			FromAccountID: accounts[0].ID,
			ToAccountID:   accounts[1].ID,
			Amount:        amount,
		},
		FromAccount: accounts[0],
		ToAccount:   accounts[1],
		FromEntry: db.Entry{
			ID:        1,
			AccountID: accounts[0].ID,
			Amount:    -amount,
		},
		ToEntry: db.Entry{
			ID:        2,
			AccountID: accounts[1].ID,
			Amount:    amount,
		},
	}

	testCases := []struct {
		name          string
		arg           CreateTransferRequest
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "Created",
			arg:  validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, accounts[0].Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[0].ID)).
					Times(1).
					Return(accounts[0], nil)
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[1].ID)).
					Times(1).
					Return(accounts[1], nil)
				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Eq(expectedArg)).
					Times(1).
					Return(expectedResult, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, recorder.Code)
				require.Equal(t, expectedResult, unmarshallTransfer(t, recorder.Body))
			},
		},
		{
			name: "Bad Request",
			arg: CreateTransferRequest{
				FromAccountID: accounts[0].ID,
				ToAccountID:   accounts[1].ID,
				Amount:        amount,
				Currency:      "AUD",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, accounts[0].Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": "Key: 'CreateTransferRequest.Currency' Error:Field validation for 'Currency' failed on the 'currency' tag"}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Not Found - FromAccount",
			arg: CreateTransferRequest{
				FromAccountID: accounts[0].ID + 1,
				ToAccountID:   accounts[0].ID,
				Amount:        amount,
				Currency:      "BRL",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, accounts[0].Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[0].ID+1)).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrNoRows.Error()}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Not Found - ToAccount",
			arg: CreateTransferRequest{
				FromAccountID: accounts[0].ID,
				ToAccountID:   accounts[1].ID + 1,
				Amount:        amount,
				Currency:      "BRL",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, accounts[0].Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[0].ID)).
					Times(1).
					Return(accounts[0], nil)
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[1].ID+1)).
					Times(1).
					Return(db.Account{}, sql.ErrNoRows)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrNoRows.Error()}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Bad Request - Different Currency",
			arg: CreateTransferRequest{
				FromAccountID: accounts[0].ID,
				ToAccountID:   accounts[1].ID,
				Amount:        amount,
				Currency:      "USD",
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, accounts[0].Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[0].ID)).
					Times(1).
					Return(accounts[0], nil)
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[1].ID)).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
			},
		},
		{
			name: "Internal Server Error - Validate From Account Transfer",
			arg:  validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, accounts[0].Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[0].ID)).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)
				store.EXPECT().GetAccount(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrConnDone.Error()}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Internal Server Error - Validate To Account Transfer",
			arg:  validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, accounts[0].Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[0].ID)).
					Times(1).
					Return(accounts[0], nil)
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[1].ID)).
					Times(1).
					Return(db.Account{}, sql.ErrConnDone)
				store.EXPECT().TransferTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrConnDone.Error()}, UnmarshallAny(t, recorder.Body))
			},
		},
		{
			name: "Internal Server Error",
			arg:  validArg,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, accounts[0].Owner, time.Minute)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[0].ID)).
					Times(1).
					Return(accounts[0], nil)
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accounts[1].ID)).
					Times(1).
					Return(accounts[1], nil)
				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Eq(expectedArg)).
					Times(1).
					Return(db.TransferTxResult{}, sql.ErrTxDone)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
				require.Exactly(t, map[string]interface{}{"error": sql.ErrTxDone.Error()}, UnmarshallAny(t, recorder.Body))
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			testCase.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/transfers")

			var buf bytes.Buffer

			err := json.NewEncoder(&buf).Encode(testCase.arg)
			require.NoError(t, err)

			request, err := http.NewRequest("POST", url, &buf)
			require.NoError(t, err)

			// setup authorization middleware
			testCase.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)

			testCase.checkResponse(t, recorder)
		})
	}
}
