package main

import (
	"context"
	"database/sql"
	_ "embed"
	"log"
	"net/http"
	"quattrinitrack/database"
	"quattrinitrack/router"

	_ "modernc.org/sqlite"
)

//go:embed database/SQL/schema.sql
var createTables string

func initDB(ctx context.Context) *sql.DB {
	db, err := sql.Open("sqlite", "db.sqlite")
	if err != nil {
		panic(err)
	}

	_, err = db.ExecContext(ctx, createTables)
	if err != nil {
		panic(err)
	}
	return db
}

func main() {
	ctx := context.Background()
	db := initDB(ctx)
	queries := database.New(db)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router.New(queries),
	}

	log.Println("Server started on localhost:8080")
	log.Fatal(server.ListenAndServe())
}
