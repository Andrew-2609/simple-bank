package api

import (
	"bytes"
	"database/sql"
	"os"
	"testing"
	"time"

	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	"github.com/Andrew-2609/simple-bank/util"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

var (
	testQueries *db.Queries
	testDB      *sql.DB
)

func newTestServer(t *testing.T, store db.Store) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	server, err := NewServer(config, store)
	require.NoError(t, err)

	return server
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func UnmarshallAny(t *testing.T, responseBody *bytes.Buffer) any {
	unmarshalledObject, err := util.UnmarshallJsonBody[any](responseBody)
	require.NoError(t, err)
	return unmarshalledObject
}
