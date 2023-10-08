package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/Andrew-2609/simple-bank/db/mock"
	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	"github.com/Andrew-2609/simple-bank/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func createRandomAccounts() [2]db.Account {
	return [2]db.Account{
		{
			ID:       util.RandomInt(1, 1000),
			Owner:    util.RandomOwner(),
			Balance:  util.RandomAmount(),
			Currency: util.BRL,
		}, {
			ID:       util.RandomInt(1, 1000),
			Owner:    util.RandomOwner(),
			Balance:  util.RandomAmount(),
			Currency: util.BRL,
		},
	}
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
			CreatedAt:     time.Now(),
		},
		FromAccount: accounts[0],
		ToAccount:   accounts[1],
		FromEntry: db.Entry{
			ID:        1,
			AccountID: accounts[0].ID,
			Amount:    -amount,
			CreatedAt: time.Now(),
		},
		ToEntry: db.Entry{
			ID:        2,
			AccountID: accounts[1].ID,
			Amount:    amount,
			CreatedAt: time.Now(),
		},
	}

	testCases := []struct {
		name          string
		arg           CreateTransferRequest
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "Created",
			arg:  validArg,
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
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			testCase.buildStubs(store)

			server := NewServer(store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/transfers")

			var buf bytes.Buffer

			err := json.NewEncoder(&buf).Encode(testCase.arg)
			require.NoError(t, err)

			request, err := http.NewRequest("POST", url, &buf)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)

			testCase.checkResponse(t, recorder)
		})
	}
}
