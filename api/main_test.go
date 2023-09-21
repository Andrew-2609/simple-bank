package api

import (
	"database/sql"
	"os"
	"testing"

	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

var (
	testQueries *db.Queries
	testDB      *sql.DB
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
