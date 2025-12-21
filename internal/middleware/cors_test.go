package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCORS(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkHeaders   func(*testing.T, http.Header)
	}{
		{
			name:           "GET request passes through with CORS headers",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, h http.Header) {
				assert.Equal(t, "*", h.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", h.Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "Origin, Content-Type, Authorization", h.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "86400", h.Get("Access-Control-Max-Age"))
			},
		},
		{
			name:           "POST request passes through with CORS headers",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, h http.Header) {
				assert.Equal(t, "*", h.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", h.Get("Access-Control-Allow-Methods"))
			},
		},
		{
			name:           "PUT request passes through with CORS headers",
			method:         http.MethodPut,
			expectedStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, h http.Header) {
				assert.Equal(t, "*", h.Get("Access-Control-Allow-Origin"))
			},
		},
		{
			name:           "DELETE request passes through with CORS headers",
			method:         http.MethodDelete,
			expectedStatus: http.StatusOK,
			checkHeaders: func(t *testing.T, h http.Header) {
				assert.Equal(t, "*", h.Get("Access-Control-Allow-Origin"))
			},
		},
		{
			name:           "OPTIONS preflight request returns 204",
			method:         http.MethodOptions,
			expectedStatus: http.StatusNoContent,
			checkHeaders: func(t *testing.T, h http.Header) {
				assert.Equal(t, "*", h.Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "GET, POST, PUT, DELETE, OPTIONS", h.Get("Access-Control-Allow-Methods"))
				assert.Equal(t, "Origin, Content-Type, Authorization", h.Get("Access-Control-Allow-Headers"))
				assert.Equal(t, "86400", h.Get("Access-Control-Max-Age"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CORS())
			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			router.POST("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			router.PUT("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			router.DELETE("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})
			router.OPTIONS("/test", func(c *gin.Context) {
				// This won't be reached due to CORS middleware aborting
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkHeaders != nil {
				tt.checkHeaders(t, w.Header())
			}
		})
	}
}

func TestCORS_PreflightDoesNotReachHandler(t *testing.T) {
	handlerCalled := false

	router := gin.New()
	router.Use(CORS())
	router.OPTIONS("/test", func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.False(t, handlerCalled, "handler should not be called for preflight requests")
}
