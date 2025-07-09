package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"quattrinitrack/database"
	"strconv"
)

type CategoryQuerier interface {
	GetAllCategories(ctx context.Context) ([]database.Category, error)
	GetCategoryByID(ctx context.Context, id int64) (database.Category, error)
	DeleteCategory(ctx context.Context, id int64) error
	InsertCategory(ctx context.Context, name string) error
}

func Category(queries CategoryQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if req.Method == http.MethodGet {
			id := req.URL.Query().Get("id")

			switch {
			case id != "":
				idNum, err := strconv.ParseInt(id, 10, 64)
				if err != nil {
					log.Printf("error in converting id")
					return
				}
				getCategoryByID(w, ctx, queries, idNum)
			default:
				getAllCategories(w, ctx, queries)
			}

		}

		if req.Method == http.MethodPost {
		}

		if req.Method == http.MethodDelete {
		}
	}
}

func getAllCategories(w http.ResponseWriter, ctx context.Context, queries CategoryQuerier) {
	categories, err := queries.GetAllCategories(ctx)
	if err != nil {
		log.Printf("error getting categories %v", err)
		http.Error(w, "Internal Sever Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(categories)
	if err != nil {
		log.Printf("error encoding categories %v", err)
		http.Error(w, "Internal Sever Error", http.StatusInternalServerError)
		return
	}
}

func getCategoryByID(w http.ResponseWriter, ctx context.Context, queries CategoryQuerier, id int64) {
	category, err := queries.GetCategoryByID(ctx, id)
	if err != nil {
		log.Printf("category not found with id: %d, error: %v", id, err)
		http.Error(w, "no category found with the given ID", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(category)
	if err != nil {
		log.Printf("error encoding category %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}
