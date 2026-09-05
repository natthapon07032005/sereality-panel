package controller

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewIndexControllerInitializesLoginRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := NewIndexController(gin.New().Group("/"))
	if controller.loginLimiter == nil {
		t.Fatal("login rate limiter was not initialized")
	}
}

func TestLoginAttemptKeyIsStableAndDoesNotContainIdentity(t *testing.T) {
	key := loginAttemptKey("203.0.113.10", "admin")
	if key != loginAttemptKey("203.0.113.10", "admin") {
		t.Fatal("login attempt key is not stable")
	}
	if key == "" || len(key) != 64 {
		t.Fatalf("login attempt key = %q, want a SHA-256 hex key", key)
	}
	if key == "admin" || key == "203.0.113.10" {
		t.Fatal("login attempt key exposed the identity")
	}
}
