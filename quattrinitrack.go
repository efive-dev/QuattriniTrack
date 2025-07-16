package main

import (
	"context"
	"database/sql"
	_ "embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"quattrinitrack/config"
	"quattrinitrack/database"
	"quattrinitrack/logger"
	"quattrinitrack/router"
	"quattrinitrack/tui"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed database/SQL/schema.sql
var createTables string

func initDB(ctx context.Context) *sql.DB {
	db, err := sql.Open("sqlite", "db.sqlite?_foreign_keys=on")
	if err != nil {
		panic(err)
	}

	if err := db.PingContext(ctx); err != nil {
		panic(err)
	}

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

	_, err = db.ExecContext(ctx, createTables)
	if err != nil {
		panic(err)
	}

	return db
}

func main() {
	logger.SetupLogCapture()

	config.LoadEnv()

	// Initialize database
	ctx := context.Background()
	db := initDB(ctx)
	defer db.Close()

	queries := database.New(db)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":8080",
		Handler: router.New(queries),
	}

	// Channel to handle server shutdown
	serverDone := make(chan error, 1)

	// Start the server in a goroutine
	go func() {
		log.Println("Server started on localhost:8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server failed to start: %v", err)
			serverDone <- err
		} else {
			serverDone <- nil
		}
	}()

	// Channel to handle graceful shutdown signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Channel to handle TUI completion
	tuiDone := make(chan error, 1)

	// Start the TUI in a goroutine
	go func() {
		tui.Init()
		tuiDone <- nil
	}()

	select {
	case err := <-tuiDone:
		log.Println("TUI shutting down...")
		if err != nil {
			log.Printf("TUI error: %v", err)
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		} else {
			log.Println("Server shutdown gracefully")
		}

	case err := <-serverDone:
		if err != nil {
			log.Printf("Server error: %v", err)
		}
		log.Println("Server stopped")

	case sig := <-quit:
		// Received shutdown signal
		log.Printf("Received signal: %v", sig)

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		} else {
			log.Println("Server shutdown gracefully")
		}
	}

	log.Println("Application shutting down...")
}
