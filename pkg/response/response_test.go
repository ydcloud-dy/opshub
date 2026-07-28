package response

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestErrorCodeUsesHTTPStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict, http.StatusInternalServerError} {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		ErrorCode(context, status, "test error")
		if recorder.Code != status {
			t.Fatalf("status %d produced HTTP %d", status, recorder.Code)
		}
	}
}
