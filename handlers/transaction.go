// Package handlers defines the functions to handle each endpoint
package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"quattrinitrack/database"
	"strconv"
)

type TransactionQuerier interface {
	GetAllTransactions(ctx context.Context) ([]database.Transaction, error)
	GetTransactionByID(ctx context.Context, id int64) (database.Transaction, error)
	GetTransactionByName(ctx context.Context, name string) ([]database.Transaction, error)
	DeleteTransaction(ctx context.Context, id int64) error
	InsertTransaction(ctx context.Context, params database.InsertTransactionParams) error
}

func Transaction(queries TransactionQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()

		if req.Method == http.MethodGet {
			id := req.URL.Query().Get("id")
			name := req.URL.Query().Get("name")
			switch {
			case id != "":
				id, err := strconv.ParseInt(id, 10, 64)
				if err != nil {
					log.Printf("error in converting id")
					return
				}
				getTransactionByID(w, ctx, queries, id)
			case name != "":
				getTransactionByName(w, ctx, queries, name)
			case id != "" && name != "":
				getAllTransactions(w, ctx, queries)

			default:
				getAllTransactions(w, ctx, queries)
			}
		}

		if req.Method == http.MethodPost {
			insertTransaction(w, req, ctx, queries)
		}

		if req.Method == http.MethodDelete {
			id := req.URL.Query().Get("id")
			idNum, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				log.Printf("error in converting id")
				return
			}
			deleteTransaction(w, ctx, queries, idNum)
		}
	}
}

func getAllTransactions(w http.ResponseWriter, ctx context.Context, queries TransactionQuerier) {
	transactions, err := queries.GetAllTransactions(ctx)
	if err != nil {
		log.Printf("error getting the transaction %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(transactions)
	if err != nil {
		log.Printf("error encoding transactions %v", err)
		http.Error(w, "Internal sever error", http.StatusInternalServerError)
		return
	}
}

func getTransactionByID(w http.ResponseWriter, ctx context.Context, queries TransactionQuerier, id int64) {
	transaction, err := queries.GetTransactionByID(ctx, id)
	if err != nil {
		log.Printf("transaction not found with id: %d, error: %v", id, err)
		http.Error(w, "No transaction found with the given ID", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(transaction)
	if err != nil {
		log.Printf("error encoding transaction %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func getTransactionByName(w http.ResponseWriter, ctx context.Context, queries TransactionQuerier, name string) {
	transactions, err := queries.GetTransactionByName(ctx, name)
	if err != nil {
		log.Printf("error getting transactions by name %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if len(transactions) == 0 {
		http.Error(w, "No transactions found with that name", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(transactions)
	if err != nil {
		log.Printf("error encoding transactions %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func insertTransaction(w http.ResponseWriter, req *http.Request, ctx context.Context, queries TransactionQuerier) {
	var transaction database.Transaction
	err := json.NewDecoder(req.Body).Decode(&transaction)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if transaction.Name == "" || transaction.Cost <= 0 || transaction.Date.IsZero() {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	err = queries.InsertTransaction(ctx, database.InsertTransactionParams{
		Name:         transaction.Name,
		Cost:         transaction.Cost,
		Date:         transaction.Date,
		CategoriesID: transaction.CategoriesID,
	})
	if err != nil {
		log.Printf("error in inserting transaction into db %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	response := map[string]string{"message": "Transaction created successfully"}
	json.NewEncoder(w).Encode(response)
}

func deleteTransaction(w http.ResponseWriter, ctx context.Context, queries TransactionQuerier, id int64) {
	_, err := queries.GetTransactionByID(ctx, id)
	if err != nil {
		log.Printf("no transaction with id %v present", id)
		http.Error(w, "Status Bad Request", http.StatusBadRequest)
		return
	}
	err = queries.DeleteTransaction(ctx, id)
	if err != nil {
		log.Printf("can not delete transaction with id %d", id)
		http.Error(w, "Status Bad Request", http.StatusBadRequest)
		return
	}
}
