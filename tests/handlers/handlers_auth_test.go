package handlers_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"quattrinitrack/database"
	"quattrinitrack/handlers"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// MockUserStore mocks the database queries for testing
type MockUserStore struct {
	mock.Mock
}

func (m *MockUserStore) CreateUser(ctx context.Context, arg database.CreateUserParams) (database.CreateUserRow, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(database.CreateUserRow), args.Error(1)
}

func (m *MockUserStore) GetUserByEmail(ctx context.Context, email string) (database.User, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(database.User), args.Error(1)
}

func TestRegister(t *testing.T) {
	mockStore := new(MockUserStore)
	handler := handlers.Register(mockStore)

	mockStore.On("CreateUser", mock.Anything, mock.AnythingOfType("database.CreateUserParams")).
		Return(database.CreateUserRow{ID: 1, Email: "test@example.com"}, nil)

	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`))
	w := httptest.NewRecorder()

	handler(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogin(t *testing.T) {
	mockStore := new(MockUserStore)
	handler := handlers.Login(mockStore)

	mockPassword := "password123"
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(mockPassword), bcrypt.DefaultCost)

	mockStore.On("GetUserByEmail", mock.Anything, "test@example.com").
		Return(database.User{ID: 1, Email: "test@example.com", PasswordHash: string(hashedPassword)}, nil)

	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(`{"email":"test@example.com","password":"password123"}`))
	w := httptest.NewRecorder()

	handler(w, req)
	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
