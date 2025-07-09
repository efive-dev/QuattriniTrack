package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
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

// GetCategoriesByID /category?id=someId
func TestGetCategoryByIDSuccess(t *testing.T) {
	_, _, expectedCategories, _ := setUpCategoriesGetTest()

	mockQueries := new(MockQueries)
	mockQueries.On("GetCategoryByID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(expectedCategories[0], nil)

	handler := handlers.Category(mockQueries)
	req := httptest.NewRequest(http.MethodGet, "/category?id=1", nil)
	w := httptest.NewRecorder()

	handler(w, req)
	var actualCategory database.Category
	err := json.NewDecoder(w.Body).Decode(&actualCategory)
	assert.NoError(t, err)
	assert.Equal(t, expectedCategories[0].ID, actualCategory.ID)
	assert.Equal(t, expectedCategories[0].Name, actualCategory.Name)
}

func TestGetCategoryByIDError(t *testing.T) {
	mockQueries := new(MockQueries)
	mockQueries.On("GetCategoryByID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(database.Category{}, sql.ErrNoRows)

	handler := handlers.Category(mockQueries)
	req := httptest.NewRequest(http.MethodGet, "/category?id=89", nil)
	w := httptest.NewRecorder()

	handler(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
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

// POST request /category
func TestCategoryPOSTHeaderAndCode(t *testing.T) {
	w, _, _, mockQueries := setUpCategoriesPostTest()
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	mockQueries.AssertExpectations(t)
}

func TestCategoryInvalidJSONName(t *testing.T) {
	_, _, _, mockQueries := setUpCategoriesPostTest()
	category := database.Category{
		Name: "",
	}
	jsonData, _ := json.Marshal(category)
	req := httptest.NewRequest(http.MethodPost, "/category", bytes.NewBuffer(jsonData))
	w := httptest.NewRecorder()
	handler := handlers.Category(mockQueries)
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid JSON")
	mockQueries.AssertExpectations(t)
}

func setUpCategoriesPostTest() (*httptest.ResponseRecorder, *http.Request, database.Category, *MockQueries) {
	category := database.Category{
		Name: "test",
	}

	mockQueries := new(MockQueries)
	mockQueries.On("InsertCategory", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("string")).Return(nil)

	handler := handlers.Category(mockQueries)
	jsonData, _ := json.Marshal(category)
	req := httptest.NewRequest("POST", "/category", bytes.NewBuffer(jsonData))
	w := httptest.NewRecorder()

	handler(w, req)

	return w, req, category, mockQueries
}

// deleteCategory DELETE /Category?id=someid
func TestDeleteCategorySuccess(t *testing.T) {
	_, _, expectedCategories, _ := setUpCategoriesGetTest()
	mockQueries := new(MockQueries)

	mockQueries.On("GetCategoryByID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(expectedCategories[0], nil)
	mockQueries.On("DeleteCategory", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(nil)

	handler := handlers.Category(mockQueries)
	req := httptest.NewRequest("DELETE", "/category/?id=1", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockQueries.AssertExpectations(t)
}

func TestDeleteCategoryNotFound(t *testing.T) {
	mockQueries := new(MockQueries)

	mockQueries.On("GetCategoryByID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(database.Category{}, sql.ErrNoRows)

	handler := handlers.Category(mockQueries)
	req := httptest.NewRequest("DELETE", "/category/?id=999", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockQueries.AssertExpectations(t)
}

func TestDeleteCategoryError(t *testing.T) {
	_, _, expectedCategories, _ := setUpCategoriesGetTest()
	mockQueries := new(MockQueries)

	mockQueries.On("GetCategoryByID", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(expectedCategories[0], nil)

	mockQueries.On("DeleteCategory", mock.AnythingOfType("context.backgroundCtx"), mock.AnythingOfType("int64")).Return(errors.New("database error"))

	handler := handlers.Category(mockQueries)
	req := httptest.NewRequest("DELETE", "/category/?id=1", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockQueries.AssertExpectations(t)
}
