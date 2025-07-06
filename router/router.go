package router

import (
	"net/http"
	"quattrinitrack/database"
	"quattrinitrack/handlers"
)

func New(queries *database.Queries) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /transaction", handlers.Transaction)
	return mux
}
