package handlers

import (
	"context"
	"quattrinitrack/database"

	"github.com/stretchr/testify/mock"
)

type MockQueries struct {
	mock.Mock
}

func (m *MockQueries) GetAllTransactions(ctx context.Context) ([]database.Transaction, error) {
	args := m.Called(ctx)
	return args.Get(0).([]database.Transaction), args.Error(1)
}

func (m *MockQueries) InsertTransaction(ctx context.Context, params database.InsertTransactionParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}
