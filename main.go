package main

import (
	"database/sql"
	"log"

	"github.com/Andrew-2609/simple-bank/api"
	db "github.com/Andrew-2609/simple-bank/db/sqlc"
	"github.com/Andrew-2609/simple-bank/util"
	_ "github.com/lib/pq"
)

func main() {
	config, err := util.LoadConfig(".")

	if err != nil {
		log.Fatalf("Could not load environment configuration: %v", err)
	}

	conn, err := sql.Open(config.DBDriver, config.DBSource)

	if err != nil {
		log.Fatalf("ERROR: could not connect to the Database: %v", err)
	}

	server, err := api.NewServer(config, db.NewSQLStore(conn))

	if err != nil {
		log.Fatalf("Could not create server: %v", err)
	}

	err = server.Start(config.ServerAddress)

	if err != nil {
		log.Fatalf("Could not start server: %v", err)
	}
}
