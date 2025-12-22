//go:build api

package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"gin-sample/test/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheck(t *testing.T) {
	t.Run("returns ok status", func(t *testing.T) {
		w := testutil.MakeRequest(t, testServer.Router, http.MethodGet, "/health", nil)

		require.Equal(t, http.StatusOK, w.Code, "health check should return 200")

		var resp map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err, "response should be valid JSON")

		assert.Equal(t, "ok", resp["status"], "status should be 'ok'")
	})
}
