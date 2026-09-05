package service_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"x-ui/web/controller"
	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

func TestBuildNodeSyncPlanReturnsDeterministicSanitizedDryRun(t *testing.T) {
	plan, err := service.BuildNodeSyncPlan(service.NodeSyncPlanInput{
		NodeID: 42,
		DryRun: true,
		Local: service.NodeSyncSnapshot{Complete: true, Items: []service.NodeSyncItem{
			{ID: "remove-z", Revision: "local-secret-value"},
			{ID: "unchanged", Revision: "same-revision"},
			{ID: "update-me", Revision: "local-revision"},
			{ID: "remove-a", Revision: "local-revision"},
		}},
		Remote: service.NodeSyncSnapshot{Complete: true, Items: []service.NodeSyncItem{
			{ID: "update-me", Revision: "remote-api-token-value"},
			{ID: "add-z", Revision: "remote-revision"},
			{ID: "unchanged", Revision: "same-revision"},
			{ID: "add-a", Revision: "remote-revision"},
		}},
	})
	if err != nil {
		t.Fatalf("BuildNodeSyncPlan() error = %v", err)
	}

	if plan.NodeID != 42 || !plan.DryRun || plan.Applied {
		t.Fatalf("plan metadata = %+v, want node 42 dry-run and not applied", plan)
	}
	if plan.Added != 2 || plan.Updated != 1 || plan.Removed != 2 {
		t.Fatalf("plan counts = %+v, want two adds, one update, and two removes", plan)
	}

	wantOperations := []service.NodeSyncOperation{
		{Action: service.NodeSyncAdd, ItemID: "add-a"},
		{Action: service.NodeSyncAdd, ItemID: "add-z"},
		{Action: service.NodeSyncUpdate, ItemID: "update-me"},
		{Action: service.NodeSyncRemove, ItemID: "remove-a"},
		{Action: service.NodeSyncRemove, ItemID: "remove-z"},
	}
	if !reflect.DeepEqual(plan.Operations, wantOperations) {
		t.Fatalf("plan operations = %#v, want %#v", plan.Operations, wantOperations)
	}

	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal(plan) error = %v", err)
	}
	for _, forbidden := range []string{
		"local-secret-value",
		"remote-api-token-value",
		"apiToken",
		"secret",
		"revision",
	} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("plan JSON leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestBuildNodeSyncPlanRejectsUnsafeSnapshotInputs(t *testing.T) {
	tests := []struct {
		name    string
		input   service.NodeSyncPlanInput
		wantErr error
	}{
		{
			name:    "missing node ID",
			input:   service.NodeSyncPlanInput{},
			wantErr: service.ErrNodeSyncNodeIDRequired,
		},
		{
			name: "blank item ID",
			input: service.NodeSyncPlanInput{
				NodeID: 1,
				Local:  service.NodeSyncSnapshot{Complete: true, Items: []service.NodeSyncItem{{ID: "  "}}},
				Remote: service.NodeSyncSnapshot{Complete: true},
			},
			wantErr: service.ErrNodeSyncItemIDRequired,
		},
		{
			name: "duplicate local item ID after normalization",
			input: service.NodeSyncPlanInput{
				NodeID: 1,
				Local: service.NodeSyncSnapshot{Complete: true, Items: []service.NodeSyncItem{
					{ID: "client-a", Revision: "one"},
					{ID: " client-a ", Revision: "two"},
				}},
				Remote: service.NodeSyncSnapshot{Complete: true},
			},
			wantErr: service.ErrNodeSyncDuplicateItemID,
		},
		{
			name: "duplicate remote item ID",
			input: service.NodeSyncPlanInput{
				NodeID: 1,
				Local:  service.NodeSyncSnapshot{Complete: true},
				Remote: service.NodeSyncSnapshot{Complete: true, Items: []service.NodeSyncItem{
					{ID: "client-a", Revision: "one"},
					{ID: "client-a", Revision: "two"},
				}},
			},
			wantErr: service.ErrNodeSyncDuplicateItemID,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.BuildNodeSyncPlan(test.input)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("BuildNodeSyncPlan() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}

func TestBuildNodeSyncPlanRejectsIncompleteSnapshots(t *testing.T) {
	_, err := service.BuildNodeSyncPlan(service.NodeSyncPlanInput{
		NodeID: 1,
		Local:  service.NodeSyncSnapshot{Complete: true},
		Remote: service.NodeSyncSnapshot{},
	})
	if !errors.Is(err, service.ErrNodeSyncSnapshotIncomplete) {
		t.Fatalf("BuildNodeSyncPlan() error = %v, want incomplete snapshot error", err)
	}
}

func TestNodeSyncControllerReturnsPlanOnlyWithoutRequestSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller.NewNodeSyncController(router.Group("/api"))

	req := httptest.NewRequest(http.MethodPost, "/api/nodes/15/sync/plan", strings.NewReader(`{
		"dryRun": true,
		"apiToken": "controller-api-token-that-must-not-return",
		"secret": "controller-secret-that-must-not-return",
		"local": {"complete": true, "items": [{"id": "client-a", "revision": "local-secret-revision"}]},
		"remote": {"complete": true, "items": [{"id": "client-b", "revision": "remote-token-revision"}]}
	}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}

	var envelope struct {
		Success bool                 `json:"success"`
		Data    service.NodeSyncPlan `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("json.Unmarshal(response) error = %v", err)
	}
	if !envelope.Success {
		t.Fatalf("controller response was not successful: %s", response.Body.String())
	}
	plan := envelope.Data
	if plan.NodeID != 15 || !plan.DryRun || plan.Applied {
		t.Fatalf("controller plan metadata = %+v, want node 15 dry-run and not applied", plan)
	}
	if plan.Added != 1 || plan.Updated != 0 || plan.Removed != 1 {
		t.Fatalf("controller plan counts = %+v, want one add and one remove", plan)
	}
	for _, forbidden := range []string{
		"controller-api-token-that-must-not-return",
		"controller-secret-that-must-not-return",
		"local-secret-revision",
		"remote-token-revision",
		"apiToken",
		"secret",
		"revision",
	} {
		if strings.Contains(response.Body.String(), forbidden) {
			t.Fatalf("controller response leaked %q: %s", forbidden, response.Body.String())
		}
	}
}

func TestNodeSyncControllerRequiresExplicitDryRunMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller.NewNodeSyncController(router.Group("/api"))
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/15/sync/plan", strings.NewReader(`{"local":{"complete":true},"remote":{"complete":true}}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if strings.Contains(response.Body.String(), "apiToken") || strings.Contains(response.Body.String(), "secret") {
		t.Fatal("invalid mode rejection leaked credentials")
	}
}
