package handlers

import (
	"bytes"
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
	w, _, _, mockQueries := setupTransactionGetTest()
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	mockQueries.AssertExpectations(t)
}

func TestTransactionGETResponse(t *testing.T) {
	var responseTransactions []database.Transaction
	w, _, expectedTransactions, mockQueries := setupTransactionGetTest()

	err := json.NewDecoder(w.Body).Decode(&responseTransactions)
	assert.NoError(t, err)
	assert.Len(t, responseTransactions, len(expectedTransactions))

	for i, expected := range expectedTransactions {
		actual := responseTransactions[i]
		assert.Equal(t, expected.Name, actual.Name, "at index %d", i)
		assert.Equal(t, expected.Cost, actual.Cost, "at index %d", i)
		assert.WithinDuration(t, expected.Date, actual.Date, time.Second, "at index %d", i)
	}

	mockQueries.AssertExpectations(t)
}

func setupTransactionGetTest() (*httptest.ResponseRecorder, *http.Request, []database.Transaction, *MockQueries) {
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

	return w, req, expectedTransactions, mockQueries
}

// POST request /transaction
func TestTransactionPOSTHeaderAndCode(t *testing.T) {
	w, _, _, mockQueries := setupTransactionPostTest()
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	mockQueries.AssertExpectations(t)
}

func TestTransactionPOSTInvalidJSONName(t *testing.T) {
	_, _, _, mockQueries := setupTransactionPostTest()
	// empty name should trigger an error
	transaction := database.Transaction{
		Name: "",
		Cost: 100.99,
		Date: time.Now(),
	}
	jsonData, _ := json.Marshal(transaction)
	req := httptest.NewRequest("POST", "/transaction", bytes.NewBuffer(jsonData))
	w := httptest.NewRecorder()
	handler := handlers.Transaction(mockQueries)
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid JSON")
	mockQueries.AssertExpectations(t)
}

func TestTransactionPOSTInvalidJSONCost(t *testing.T) {
	_, _, _, mockQueries := setupTransactionPostTest()
	// negative cost should trigger an error
	transaction := database.Transaction{
		Name: "A name",
		Cost: -3,
		Date: time.Now(),
	}
	jsonData, _ := json.Marshal(transaction)
	req := httptest.NewRequest("POST", "/transaction", bytes.NewBuffer(jsonData))
	w := httptest.NewRecorder()
	handler := handlers.Transaction(mockQueries)
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid JSON")
	mockQueries.AssertExpectations(t)
}

func TestTransactionPOSTInvalidJSONDate(t *testing.T) {
	_, _, _, mockQueries := setupTransactionPostTest()
	// Not handled date should trigger an error
	transaction := database.Transaction{
		Name: "A name",
		Cost: 3,
	}
	jsonData, _ := json.Marshal(transaction)
	req := httptest.NewRequest("POST", "/transaction", bytes.NewBuffer(jsonData))
	w := httptest.NewRecorder()
	handler := handlers.Transaction(mockQueries)
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid JSON")
	mockQueries.AssertExpectations(t)
}

func setupTransactionPostTest() (*httptest.ResponseRecorder, *http.Request, database.Transaction, *MockQueries) {
	transaction := database.Transaction{
		Name: "test",
		Cost: 100.99,
		Date: time.Now(),
	}

	mockQueries := new(MockQueries)
	mockQueries.On("InsertTransaction", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("database.InsertTransactionParams")).Return(nil)

	handler := handlers.Transaction(mockQueries)
	jsonData, _ := json.Marshal(transaction)
	req := httptest.NewRequest("POST", "/transaction", bytes.NewBuffer(jsonData))
	w := httptest.NewRecorder()

	handler(w, req)

	return w, req, transaction, mockQueries
}
