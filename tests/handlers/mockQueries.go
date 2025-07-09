package handlers

import (
	"context"
	"quattrinitrack/database"

	"github.com/stretchr/testify/mock"
)

type MockQueries struct {
	mock.Mock
}

// Transactions

func (m *MockQueries) GetAllTransactions(ctx context.Context) ([]database.Transaction, error) {
	args := m.Called(ctx)
	return args.Get(0).([]database.Transaction), args.Error(1)
}

func (m *MockQueries) GetTransactionByID(ctx context.Context, id int64) (database.Transaction, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(database.Transaction), args.Error(1)
}

func (m *MockQueries) GetTransactionByName(ctx context.Context, name string) ([]database.Transaction, error) {
	args := m.Called(ctx, name)
	return args.Get(0).([]database.Transaction), args.Error(1)
}

func (m *MockQueries) DeleteTransaction(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueries) InsertTransaction(ctx context.Context, params database.InsertTransactionParams) error {
	args := m.Called(ctx, params)
	return args.Error(0)
}

// Categories

func (m *MockQueries) GetAllCategories(ctx context.Context) ([]database.Category, error) {
	args := m.Called(ctx)
	return args.Get(0).([]database.Category), args.Error(1)
}

func (m *MockQueries) GetCategoryByID(ctx context.Context, id int64) (database.Category, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(database.Category), args.Error(1)
}

func (m *MockQueries) DeleteCategory(ctx context.Context, id int64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockQueries) InsertCategory(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}
