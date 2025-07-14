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
	// Setup log capture system first
	logger.SetupLogCapture()

	// Load configuration
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
		// Initialize and run the TUI (this will block until the user quits)
		tui.Init()
		tuiDone <- nil
	}()

	// Wait for either TUI to quit, server to fail, or shutdown signal
	select {
	case err := <-tuiDone:
		// TUI quit, shutdown server gracefully
		log.Println("TUI shutting down...")
		if err != nil {
			log.Printf("TUI error: %v", err)
		}

		// Graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		} else {
			log.Println("Server shutdown gracefully")
		}

	case err := <-serverDone:
		// Server failed or stopped
		if err != nil {
			log.Printf("Server error: %v", err)
		}
		log.Println("Server stopped")

	case sig := <-quit:
		// Received shutdown signal
		log.Printf("Received signal: %v", sig)

		// Graceful shutdown with timeout
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		} else {
			log.Println("Server shutdown gracefully")
		}
	}

	// Final cleanup
	log.Println("Application shutting down...")
}
