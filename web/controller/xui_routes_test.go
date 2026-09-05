package controller

import (
	"testing"

	"github.com/gin-gonic/gin"
)

func TestXUIControllerRegistersSerealityManagementRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewXUIController(engine.Group(""))

	want := map[string]bool{
		"GET /panel/package-manager":                 false,
		"GET /panel/user-manager":                    false,
		"GET /panel/subscription-manager":            false,
		"GET /panel/node-manager":                    false,
		"GET /panel/api/v2/health":                   false,
		"POST /panel/api/v2/nodes/:nodeId/sync/plan": false,
		"POST /panel/api/v2/subscriptions":           false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Fatalf("route %s was not registered", route)
		}
	}
}

func TestXUIControllerRegistersAPIV2UserManagementRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	NewXUIController(engine.Group(""))

	want := map[string]bool{
		"GET /panel/api/v2/users":              false,
		"POST /panel/api/v2/users/:id/status":  false,
		"POST /panel/api/v2/users/:id/package": false,
	}
	for _, route := range engine.Routes() {
		key := route.Method + " " + route.Path
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for route, found := range want {
		if !found {
			t.Fatalf("route %s was not registered", route)
		}
	}
}
