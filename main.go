package main

import (
	"database/sql"
	"log"

	"github.com/Andrew-2609/simple-bank/api"
	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	_ "github.com/lib/pq"
)

const (
	dbDriver      = "postgres"
	dbSource      = "postgresql://root:secret@localhost:5432/simple_bank?sslmode=disable"
	serverAddress = "localhost:8080"
)

func main() {
	conn, err := sql.Open(dbDriver, dbSource)

	if err != nil {
		log.Fatalf("ERROR: could not connect to the Database: %v", err)
	}

	store := db.NewStore(conn)

	err = api.NewServer(store).Start(serverAddress)

	if err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
