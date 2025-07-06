package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"quattrinitrack/database"
)

type TransactionQuerier interface {
	GetAllTransactions(ctx context.Context) ([]database.Transaction, error)
	InsertTransaction(ctx context.Context, params database.InsertTransactionParams) error
}

func Transaction(queries TransactionQuerier) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()
		if request.Method == "GET" {
			transactions, err := queries.GetAllTransactions(ctx)
			if err != nil {
				log.Printf("error getting the transaction %v", err)
				http.Error(writer, "Internal server error", http.StatusInternalServerError)
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			err = json.NewEncoder(writer).Encode(transactions)
			if err != nil {
				log.Printf("error encoding transactions %v", err)
				http.Error(writer, "Internal sever error", http.StatusInternalServerError)
				return
			}
		}

		if request.Method == "POST" {
			var transaction database.Transaction
			err := json.NewDecoder(request.Body).Decode(&transaction)
			if err != nil {
				http.Error(writer, "invalid JSON", http.StatusBadRequest)
				return
			}

			if transaction.Name == "" || transaction.Cost == 0 || transaction.Date.IsZero() {
				http.Error(writer, "invalid JSON", http.StatusBadRequest)
				return
			}

			err = queries.InsertTransaction(ctx, database.InsertTransactionParams{
				Name: transaction.Name,
				Cost: transaction.Cost,
				Date: transaction.Date,
			})
			if err != nil {
				log.Printf("error in inserting transaction into db %v", err)
				http.Error(writer, "Internal server error", http.StatusInternalServerError)
				return
			}

			writer.Header().Set("Content-Type", "application/json")
			writer.WriteHeader(http.StatusCreated)
			response := map[string]string{"message": "Transaction created successfully"}
			json.NewEncoder(writer).Encode(response)
		}
	}
}
