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
	db, err := sql.Open("sqlite", "db.sqlite?_foreign_keys=on")
	if err != nil {
		panic(err)
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		panic(err)
	}

	// Explicitly enable foreign keys (double check)
	_, err = db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	if err != nil {
		panic(err)
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	err = db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		panic(err)
	}
	if fkEnabled != 1 {
		panic("Foreign key constraints are not enabled")
	}

	// Create tables (make sure categories table is created first)
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
