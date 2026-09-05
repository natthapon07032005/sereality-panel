package controller_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"x-ui/database"
	"x-ui/database/model"
	"x-ui/web/controller"
	"x-ui/web/service"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const nodeSyncControllerTestToken = "node-sync-controller-test-token"

func TestNodeSyncControllerApplyFetchesAuthenticatedCompleteRemoteSnapshot(t *testing.T) {
	var requests atomic.Int32
	remote := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Method != http.MethodGet {
			t.Errorf("remote method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/panel/api/v2/nodes/sync/snapshot" {
			t.Errorf("remote path = %s, want snapshot endpoint", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+nodeSyncControllerTestToken {
			t.Error("remote request did not use the supplied API token as bearer authentication")
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read remote request body: %v", err)
		}
		if len(body) != 0 {
			t.Error("remote snapshot request unexpectedly included a body")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"complete":true,"items":[{"id":"shared","revision":"same"},{"id":"remote-only","revision":"remote-response-revision"}]}}`))
	}))
	defer remote.Close()
	useNodeSyncTestTransport(t, remote)
	initNodeSyncControllerTestDB(t)
	node := createNodeSyncControllerTestNode(t, remote.URL+"/panel", true, nodeSyncControllerTestToken)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller.NewNodeSyncController(router.Group("/api"))
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/"+strconv.Itoa(node.ID)+"/sync/plan", strings.NewReader(`{
		"dryRun": false,
		"apiToken": "node-sync-controller-test-token",
		"local": {"complete": true, "items": [
			{"id":"shared","revision":"same"},
			{"id":"local-only","revision":"local-request-revision"}
		]},
		"remote": {"complete": true, "items": [
			{"id":"untrusted-request-item","revision":"untrusted-request-revision"}
		]}
	}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if requests.Load() != 1 {
		t.Fatalf("remote requests = %d, want 1", requests.Load())
	}

	var envelope struct {
		Success bool                 `json:"success"`
		Data    service.NodeSyncPlan `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !envelope.Success {
		t.Fatal("apply response was not successful")
	}
	if envelope.Data.DryRun || envelope.Data.Applied {
		t.Fatal("authenticated apply must return the non-destructive remote-backed plan")
	}
	if envelope.Data.Added != 1 || envelope.Data.Updated != 0 || envelope.Data.Removed != 1 {
		t.Fatalf("plan counts = %+v, want one add and one remove", envelope.Data)
	}
	if got, want := envelope.Data.Operations, []service.NodeSyncOperation{
		{Action: service.NodeSyncAdd, ItemID: "remote-only"},
		{Action: service.NodeSyncRemove, ItemID: "local-only"},
	}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("plan operations = %#v, want %#v", got, want)
	}
	assertNodeSyncControllerResponseDoesNotLeak(t, response.Body.String(), nodeSyncControllerTestToken, "local-request-revision", "untrusted-request-revision", "remote-response-revision")
}

func TestNodeSyncControllerApplyRejectsUnsafeConfiguredNodeBeforeContact(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		enabled bool
	}{
		{name: "disabled node", baseURL: "https://node.example.test/panel", enabled: false},
		{name: "non HTTPS node", baseURL: "http://node.example.test/panel", enabled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			initNodeSyncControllerTestDB(t)
			node := createNodeSyncControllerTestNode(t, test.baseURL, test.enabled, nodeSyncControllerTestToken)
			router := newNodeSyncControllerTestRouter()
			response := performNodeSyncApplyRequest(t, router, node.ID, nodeSyncControllerTestToken)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
			}
			assertNodeSyncControllerResponseDoesNotLeak(t, response.Body.String(), nodeSyncControllerTestToken)
		})
	}
}

func TestNodeSyncControllerApplyRejectsMalformedOrIncompleteRemoteSnapshotWithoutLeakingSecrets(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		secret     string
	}{
		{name: "non success status", statusCode: http.StatusServiceUnavailable, body: `{"error":"remote-server-secret"}`, secret: "remote-server-secret"},
		{name: "malformed JSON", statusCode: http.StatusOK, body: `not-json`, secret: "not-json"},
		{name: "incomplete snapshot", statusCode: http.StatusOK, body: `{"success":true,"data":{"complete":false,"items":[{"id":"hidden","revision":"remote-incomplete-secret"}]}}`, secret: "remote-incomplete-secret"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			remote := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer "+nodeSyncControllerTestToken {
					t.Error("remote request did not use bearer authentication")
				}
				w.WriteHeader(test.statusCode)
				_, _ = w.Write([]byte(test.body))
			}))
			defer remote.Close()
			useNodeSyncTestTransport(t, remote)
			initNodeSyncControllerTestDB(t)
			node := createNodeSyncControllerTestNode(t, remote.URL, true, nodeSyncControllerTestToken)
			router := newNodeSyncControllerTestRouter()
			response := performNodeSyncApplyRequest(t, router, node.ID, nodeSyncControllerTestToken)

			if response.Code != http.StatusBadGateway {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusBadGateway)
			}
			assertNodeSyncControllerResponseDoesNotLeak(t, response.Body.String(), nodeSyncControllerTestToken, test.secret)
		})
	}
}

func newNodeSyncControllerTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	controller.NewNodeSyncController(router.Group("/api"))
	return router
}

func performNodeSyncApplyRequest(t *testing.T, router http.Handler, nodeID int, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/"+strconv.Itoa(nodeID)+"/sync/plan", strings.NewReader(`{
		"dryRun": false,
		"apiToken": "`+token+`",
		"local": {"complete": true, "items": [{"id":"local","revision":"local-secret-revision"}]}
	}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	return response
}

func initNodeSyncControllerTestDB(t *testing.T) {
	t.Helper()
	_ = database.CloseDB()
	if err := database.InitDB(filepath.Join(t.TempDir(), "x-ui.db")); err != nil {
		if strings.Contains(err.Error(), "CGO_ENABLED=0") {
			t.Skip("SQLite integration requires CGO")
		}
		t.Fatalf("initialize test database: %v", err)
	}
	t.Cleanup(func() {
		_ = database.CloseDB()
	})
}

func createNodeSyncControllerTestNode(t *testing.T, baseURL string, enabled bool, token string) model.Node {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash node API token: %v", err)
	}
	node := model.Node{Name: "Remote", BaseURL: baseURL, APITokenHash: string(hash), Enabled: enabled}
	if err := database.GetDB().Create(&node).Error; err != nil {
		t.Fatalf("create node: %v", err)
	}
	return node
}

func useNodeSyncTestTransport(t *testing.T, remote *httptest.Server) {
	t.Helper()
	previous := http.DefaultTransport
	http.DefaultTransport = remote.Client().Transport
	t.Cleanup(func() {
		http.DefaultTransport = previous
	})
}

func assertNodeSyncControllerResponseDoesNotLeak(t *testing.T, body string, secrets ...string) {
	t.Helper()
	for _, secret := range secrets {
		if strings.Contains(body, secret) {
			t.Fatal("node sync response leaked a secret")
		}
	}
	if strings.Contains(body, "apiToken") || strings.Contains(body, "revision") {
		t.Fatal("node sync response exposed a sensitive field")
	}
}
