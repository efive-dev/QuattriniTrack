package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"quattrinitrack/database"
	"quattrinitrack/handlers"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	// Disable logging for all tests
	log.SetOutput(io.Discard)

	// Run tests
	code := m.Run()

	// Restore logging
	log.SetOutput(os.Stderr)

	os.Exit(code)
}

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

// getTransactionByID /transaction/?id=someid
func TestGetTransactionByIDSuccess(t *testing.T) {
	var actualTransaction database.Transaction
	_, _, expectedTransactions, _ := setupTransactionGetTest()

	mockQueries := new(MockQueries)
	mockQueries.On("GetTransactionByID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(expectedTransactions[0], nil)

	handler := handlers.Transaction(mockQueries)
	req := httptest.NewRequest("GET", "/transaction/?id=1", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	err := json.NewDecoder(w.Body).Decode(&actualTransaction)
	assert.NoError(t, err)

	assert.Equal(t, expectedTransactions[0].ID, actualTransaction.ID)
	assert.Equal(t, expectedTransactions[0].Name, actualTransaction.Name)
	assert.Equal(t, expectedTransactions[0].Cost, actualTransaction.Cost)
	assert.WithinDuration(t, expectedTransactions[0].Date, actualTransaction.Date, time.Second)
}

func TestGetTransactionByIDError(t *testing.T) {
	mockQueries := new(MockQueries)
	mockQueries.On("GetTransactionByID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(database.Transaction{}, sql.ErrNoRows)

	handler := handlers.Transaction(mockQueries)
	req := httptest.NewRequest("GET", "/transaction/?id=89", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// getTransactionByName /transaction/?name=someName
func TestGetTransactionByNameSuccess(t *testing.T) {
	var actualTransaction []database.Transaction
	_, _, expectedTransactions, _ := setupTransactionGetTest()

	mockQueries := new(MockQueries)
	mockQueries.On("GetTransactionByName", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).Return(expectedTransactions, nil)

	handler := handlers.Transaction(mockQueries)
	req := httptest.NewRequest("GET", "/transaction/?name=Coffee", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	err := json.NewDecoder(w.Body).Decode(&actualTransaction)
	assert.NoError(t, err)

	assert.Equal(t, expectedTransactions[0].ID, actualTransaction[0].ID)
	assert.Equal(t, expectedTransactions[0].Name, actualTransaction[0].Name)
	assert.Equal(t, expectedTransactions[0].Cost, actualTransaction[0].Cost)
	assert.WithinDuration(t, expectedTransactions[0].Date, actualTransaction[0].Date, time.Second)
}

func TestGetTransactionByNameError(t *testing.T) {
	mockQueries := new(MockQueries)
	// Return empty slice for "not found"
	mockQueries.On("GetTransactionByName", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).Return([]database.Transaction{}, nil)

	handler := handlers.Transaction(mockQueries)
	req := httptest.NewRequest("GET", "/transaction/?name=NonExistent", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
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

// deleteTransaction DELETE /transaction/?id=someid
func TestDeleteTransactionSuccess(t *testing.T) {
	_, _, expectedTransactions, _ := setupTransactionGetTest()
	mockQueries := new(MockQueries)

	mockQueries.On("GetTransactionByID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(expectedTransactions[0], nil)
	mockQueries.On("DeleteTransaction", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(nil)

	handler := handlers.Transaction(mockQueries)
	req := httptest.NewRequest("DELETE", "/transaction/?id=1", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockQueries.AssertExpectations(t)
}

func TestDeleteTransactionNotFound(t *testing.T) {
	mockQueries := new(MockQueries)

	mockQueries.On("GetTransactionByID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(database.Transaction{}, sql.ErrNoRows)

	handler := handlers.Transaction(mockQueries)
	req := httptest.NewRequest("DELETE", "/transaction/?id=999", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockQueries.AssertExpectations(t)
}

func TestDeleteTransactionDeleteError(t *testing.T) {
	_, _, expectedTransactions, _ := setupTransactionGetTest()
	mockQueries := new(MockQueries)

	mockQueries.On("GetTransactionByID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(expectedTransactions[0], nil)

	mockQueries.On("DeleteTransaction", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(errors.New("database error"))

	handler := handlers.Transaction(mockQueries)
	req := httptest.NewRequest("DELETE", "/transaction/?id=1", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockQueries.AssertExpectations(t)
}
