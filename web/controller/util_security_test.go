package controller

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestJSONMessageDoesNotExposeInternalErrorDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		jsonMsg(c, "load failed", errors.New("database path=/private/panel.db password=secret-value"))
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest("GET", "/", nil))

	body := response.Body.String()
	for _, forbidden := range []string{"/private/panel.db", "password=secret-value"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("response leaked internal detail %q: %s", forbidden, body)
		}
	}
}
