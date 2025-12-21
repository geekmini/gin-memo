package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

func TestSuccess(t *testing.T) {
	c, w := setupTestContext()

	data := map[string]string{"message": "hello"}
	Success(c, data)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.Empty(t, resp.Error)
}

func TestCreated(t *testing.T) {
	c, w := setupTestContext()

	data := map[string]string{"id": "123"}
	Created(c, data)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Data)
	assert.Empty(t, resp.Error)
}

func TestNoContent(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		NoContent(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.String())
}

func TestError(t *testing.T) {
	c, w := setupTestContext()

	Error(c, http.StatusTeapot, "I'm a teapot")

	assert.Equal(t, http.StatusTeapot, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Nil(t, resp.Data)
	assert.Equal(t, "I'm a teapot", resp.Error)
}

func TestBadRequest(t *testing.T) {
	c, w := setupTestContext()

	BadRequest(c, "invalid input")

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "invalid input", resp.Error)
}

func TestUnauthorized(t *testing.T) {
	c, w := setupTestContext()

	Unauthorized(c, "not authenticated")

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "not authenticated", resp.Error)
}

func TestForbidden(t *testing.T) {
	c, w := setupTestContext()

	Forbidden(c, "access denied")

	assert.Equal(t, http.StatusForbidden, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "access denied", resp.Error)
}

func TestNotFound(t *testing.T) {
	c, w := setupTestContext()

	NotFound(c, "resource not found")

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "resource not found", resp.Error)
}

func TestConflict(t *testing.T) {
	c, w := setupTestContext()

	Conflict(c, "resource already exists")

	assert.Equal(t, http.StatusConflict, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "resource already exists", resp.Error)
}

func TestInternalError(t *testing.T) {
	c, w := setupTestContext()

	InternalError(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Equal(t, "internal server error", resp.Error)
}

func TestResponseJSONSerialization(t *testing.T) {
	tests := []struct {
		name     string
		response Response
		expected string
	}{
		{
			name: "success with data",
			response: Response{
				Success: true,
				Data:    map[string]string{"key": "value"},
			},
			expected: `{"success":true,"data":{"key":"value"}}`,
		},
		{
			name: "error response",
			response: Response{
				Success: false,
				Error:   "something went wrong",
			},
			expected: `{"success":false,"error":"something went wrong"}`,
		},
		{
			name: "success without data",
			response: Response{
				Success: true,
			},
			expected: `{"success":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			assert.NoError(t, err)
			assert.JSONEq(t, tt.expected, string(data))
		})
	}
}
