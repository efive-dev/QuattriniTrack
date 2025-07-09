package handlers

import (
	"context"
	"net/http"
	"quattrinitrack/database"
)

type CategoryQuerier interface {
	GetAllCategories(ctx context.Context) ([]database.Category, error)
	GetCategoryByID(ctx context.Context, id int64) (database.Category, error)
	DeleteCategory(ctx context.Context, id int64) error
	InsertCategory(ctx context.Context, name string) error
}

func Category(queries CategoryQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {}
}
