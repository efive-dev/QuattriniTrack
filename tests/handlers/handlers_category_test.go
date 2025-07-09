package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"quattrinitrack/database"
	"quattrinitrack/handlers"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// GET request /category
func TestGetAllCategoriesSuccess(t *testing.T) {
	var actualCategories []database.Category
	w, _, expectedCategories, mockQueries := setUpCategoriesGetTest()

	err := json.NewDecoder(w.Body).Decode(&actualCategories)
	assert.NoError(t, err)
	assert.Len(t, actualCategories, len(expectedCategories))

	for i, expected := range expectedCategories {
		assert.Equal(t, expected.ID, actualCategories[i].ID, "at index %d", i)
		assert.Equal(t, expected.Name, actualCategories[i].Name, "at index %d", i)
	}
	mockQueries.AssertExpectations(t)
}

func setUpCategoriesGetTest() (*httptest.ResponseRecorder, *http.Request, []database.Category, *MockQueries) {
	expectedCategories := []database.Category{
		{Name: "Sport"},
		{Name: "Food"},
	}

	mockQueries := new(MockQueries)
	mockQueries.On("GetAllCategories", mock.AnythingOfType("context.backgroundCtx")).Return(expectedCategories, nil)

	handler := handlers.Category(mockQueries)
	req := httptest.NewRequest(http.MethodGet, "/category", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	return w, req, expectedCategories, mockQueries
}
