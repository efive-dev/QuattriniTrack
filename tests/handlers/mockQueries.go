package handlers

import (
	"context"
	"errors"
	"quattrinitrack/database"
)

type MockQueries struct {
	shouldError  bool
	transactions []database.Transaction
}

func (m *MockQueries) GetAllTransactions(ctx context.Context) ([]database.Transaction, error) {
	if m.shouldError {
		return nil, errors.New("db error")
	}
	return m.transactions, nil
}

func (m *MockQueries) InsertTransaction(ctx context.Context, params database.InsertTransactionParams) error {
	if m.shouldError {
		return errors.New("db error")
	}
	return nil
}
