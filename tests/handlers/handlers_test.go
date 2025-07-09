package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"quattrinitrack/database"
	"quattrinitrack/handlers"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// GET request /transaction
func TestTransactionGETHeaderAndCode(t *testing.T) {
	w, _, _ := setupTransactionGetTest()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestTransactionGETResponse(t *testing.T) {
	var responseTransactions []database.Transaction
	w, _, expectedTransactions := setupTransactionGetTest()

	err := json.NewDecoder(w.Body).Decode(&responseTransactions)
	assert.NoError(t, err)
	assert.Len(t, responseTransactions, len(expectedTransactions))

	for i, expected := range expectedTransactions {
		actual := responseTransactions[i]
		assert.Equal(t, expected.Name, actual.Name, "at index %d", i)
		assert.Equal(t, expected.Cost, actual.Cost, "at index %d", i)
		assert.WithinDuration(t, expected.Date, actual.Date, time.Second, "at index %d", i)
	}
}

func setupTransactionGetTest() (*httptest.ResponseRecorder, *http.Request, []database.Transaction) {
	expectedTransactions := []database.Transaction{
		{Name: "Coffee", Cost: 5.50, Date: time.Now()},
		{Name: "Hamburger", Cost: 12.00, Date: time.Now()},
	}

	mockQueries := new(MockQueries)
	mockQueries.On("GetAllTransactions", mock.AnythingOfType("context.backgroundCtx")).Return(expectedTransactions, nil)

	handler := handlers.Transaction(mockQueries)
	req := httptest.NewRequest("GET", "/transaction", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	return w, req, expectedTransactions
}
