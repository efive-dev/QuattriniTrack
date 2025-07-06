package handlers

import (
	"net/http"
	"net/http/httptest"
	"quattrinitrack/database"
	"quattrinitrack/handlers"
	"testing"
	"time"
)

func TestTransactionGETSuccess(t *testing.T) {
	mock := &MockQueries{
		transactions: []database.Transaction{
			{Name: "Coffee", Cost: 5.50, Date: time.Now()},
		},
	}

	handler := handlers.Transaction(mock)
	req := httptest.NewRequest("GET", "/transaction", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Error("want application/json content type")
	}
}

func TestTransactionGETFaiulure(t *testing.T) {
	mock := &MockQueries{shouldError: true}

	handler := handlers.Transaction(mock)
	req := httptest.NewRequest("GET", "/transaction", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("request should return 500 code, got %v instead", http.StatusInternalServerError)
	}
}
