package db

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/Andrew-2609/simple-bank/util"
	_ "github.com/lib/pq"
)

var testQueries *Queries
var testDB *sql.DB

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..")

	if err != nil {
		log.Fatalf("Could not load environment configuration: %v", err)
	}

	testDB, err = sql.Open(config.DBDriver, config.DBSource)

	if err != nil {
		log.Fatalf("ERROR: could not connect to the Database: %v", err)
	}

	testQueries = New(testDB)

	os.Exit(m.Run())
}
