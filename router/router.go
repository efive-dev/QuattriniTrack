// Package router defines the endpoints and the type of requests for those endpoints
package router

import (
	"net/http"
	"quattrinitrack/database"
	"quattrinitrack/handlers"
	middleware "quattrinitrack/middlewares"
)

func New(queries *database.Queries) http.Handler {
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("POST /register", handlers.Register(queries))
	mux.HandleFunc("POST /login", handlers.Login(queries))

	// Protected routes
	protected := http.NewServeMux()

	protected.HandleFunc("GET /transaction", handlers.Transaction(queries))
	protected.HandleFunc("POST /transaction", handlers.Transaction(queries))
	protected.HandleFunc("DELETE /transaction", handlers.Transaction(queries))

	protected.HandleFunc("GET /category", handlers.Category(queries))
	protected.HandleFunc("POST /category", handlers.Category(queries))
	protected.HandleFunc("DELETE /category", handlers.Category(queries))

	protected.HandleFunc("GET /me", handlers.Me(queries))

	// Mount protected routes under middleware
	mux.Handle("/", middleware.AuthMiddleware(protected.ServeHTTP))

	return mux
}
