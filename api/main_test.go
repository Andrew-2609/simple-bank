package api

import (
	"bytes"
	"database/sql"
	"os"
	"testing"

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

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func UnmarshallAny(t *testing.T, responseBody *bytes.Buffer) any {
	unmarshalledObject, err := util.UnmarshallJsonBody[any](responseBody)
	require.NoError(t, err)
	return unmarshalledObject
}
