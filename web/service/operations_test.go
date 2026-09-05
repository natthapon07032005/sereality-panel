package service_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	operationscontroller "x-ui/web/controller"
	"x-ui/web/service"

	"github.com/gin-gonic/gin"
)

func TestDecideRateLimitAllowsAttemptsBelowLimit(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	decision := service.DecideRateLimit(
		[]time.Time{now.Add(-10 * time.Second)},
		now,
		2,
		time.Minute,
	)

	if !decision.Allowed {
		t.Fatal("an attempt below the limit was denied")
	}
	if decision.Limit != 2 || decision.Remaining != 0 {
		t.Fatalf("decision = %+v, want limit 2 and no remaining attempts after accepting this one", decision)
	}
	if decision.RetryAfter != 0 {
		t.Fatalf("retry-after = %s, want zero for an allowed attempt", decision.RetryAfter)
	}
}

func TestDecideRateLimitExpiresAttemptsAtWindowBoundary(t *testing.T) {
	start := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	decision := service.DecideRateLimit(
		[]time.Time{start},
		start.Add(time.Minute),
		1,
		time.Minute,
	)

	if !decision.Allowed {
		t.Fatal("an attempt at the window boundary was not expired")
	}
	if decision.Remaining != 0 {
		t.Fatalf("remaining = %d, want zero after accepting the new attempt", decision.Remaining)
	}
}

func TestRateLimiterBoundsTrackedKeysByEvictingTheOldestBucket(t *testing.T) {
	limiter := service.NewRateLimiter(1, time.Minute, 2)
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)

	if decision := limiter.AllowAt("first", now); !decision.Allowed {
		t.Fatalf("first key decision = %+v, want allowed", decision)
	}
	if decision := limiter.AllowAt("second", now.Add(time.Second)); !decision.Allowed {
		t.Fatalf("second key decision = %+v, want allowed", decision)
	}
	if decision := limiter.AllowAt("third", now.Add(2*time.Second)); !decision.Allowed {
		t.Fatalf("third key decision = %+v, want allowed after bounded eviction", decision)
	}

	// The oldest bucket was evicted to keep the in-memory key set bounded.
	if decision := limiter.AllowAt("first", now.Add(3*time.Second)); !decision.Allowed {
		t.Fatalf("evicted key decision = %+v, want a fresh bucket", decision)
	}
	if got := limiter.TrackedKeyCount(); got != 2 {
		t.Fatalf("tracked key count = %d, want the configured bound of 2", got)
	}
}

func TestRateLimiterAllowUsesCurrentTime(t *testing.T) {
	limiter := service.NewRateLimiter(1, time.Minute, 1)
	if decision := limiter.Allow("current-key"); !decision.Allowed {
		t.Fatalf("first wall-clock decision = %+v, want allowed", decision)
	}
	if decision := limiter.Allow("current-key"); decision.Allowed {
		t.Fatalf("second wall-clock decision = %+v, want denied", decision)
	}
}

func TestNewAuditEventDoesNotExposeCredentialOrTokenFields(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	event := service.NewAuditEvent(service.AuditEventInput{
		Action:    "login",
		Username:  "admin",
		Resource:  "panel",
		RemoteIP:  "203.0.113.10",
		RequestID: "request-1",
		Success:   false,
		Timestamp: now,
	})

	if event.Action != "login" || event.Username != "admin" || event.RemoteIP != "203.0.113.10" || event.Success {
		t.Fatalf("audit event = %+v, want the safe event fields preserved", event)
	}

	typeOfEvent := reflect.TypeOf(event)
	for index := 0; index < typeOfEvent.NumField(); index++ {
		fieldName := strings.ToLower(typeOfEvent.Field(index).Name)
		if strings.Contains(fieldName, "password") || strings.Contains(fieldName, "secret") || strings.Contains(fieldName, "token") {
			t.Fatalf("audit event has a sensitive field: %s", typeOfEvent.Field(index).Name)
		}
	}

	payload := mustMarshalJSON(t, event)
	for _, forbidden := range []string{"password", "loginSecret", "token", "secret-token", "password-value"} {
		if strings.Contains(strings.ToLower(payload), strings.ToLower(forbidden)) {
			t.Fatalf("audit JSON contains forbidden value %q: %s", forbidden, payload)
		}
	}
}

func TestBuildBackupStatusReportsPathsAndStatusWithoutContents(t *testing.T) {
	status := service.BuildBackupStatus(service.BackupStatusInput{
		SourcePath:      "/var/lib/sereality/panel.db",
		DestinationPath: "/var/backups/panel.db",
		Status:          service.BackupStatusSucceeded,
		StartedAt:       time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC),
		CompletedAt:     time.Date(2026, 9, 2, 12, 0, 2, 0, time.UTC),
	})

	if status.SourcePath != "/var/lib/sereality/panel.db" || status.DestinationPath != "/var/backups/panel.db" {
		t.Fatalf("backup paths = %+v, want source and destination paths", status)
	}
	if status.Status != service.BackupStatusSucceeded {
		t.Fatalf("backup status = %q, want %q", status.Status, service.BackupStatusSucceeded)
	}

	typeOfStatus := reflect.TypeOf(status)
	for index := 0; index < typeOfStatus.NumField(); index++ {
		fieldName := strings.ToLower(typeOfStatus.Field(index).Name)
		if strings.Contains(fieldName, "content") || strings.Contains(fieldName, "body") || fieldName == "data" {
			t.Fatalf("backup status has a content-bearing field: %s", typeOfStatus.Field(index).Name)
		}
	}

	payload := mustMarshalJSON(t, status)
	for _, forbidden := range []string{"SQLite format 3", "database-content", "fileContents"} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("backup JSON contains file content marker %q: %s", forbidden, payload)
		}
	}
}

func TestOperationsControllerWritesSafeOperationalDTOs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controller := operationscontroller.NewOperationsController()

	tests := []struct {
		name     string
		write    func(*gin.Context)
		wantKey  string
		wantJSON string
	}{
		{
			name:     "rate limit decision",
			wantKey:  "allowed",
			wantJSON: "false",
			write: func(c *gin.Context) {
				controller.WriteRateLimitDecision(c, service.RateLimitDecision{Allowed: false, Limit: 3, Remaining: 0, RetryAfter: 10 * time.Second})
			},
		},
		{
			name:     "audit event",
			wantKey:  "action",
			wantJSON: `"login"`,
			write: func(c *gin.Context) {
				controller.WriteAuditEvent(c, service.NewAuditEvent(service.AuditEventInput{Action: "login", Username: "admin", Success: true}))
			},
		},
		{
			name:     "backup status",
			wantKey:  "status",
			wantJSON: `"failed"`,
			write: func(c *gin.Context) {
				controller.WriteBackupStatus(c, service.BuildBackupStatus(service.BackupStatusInput{Status: service.BackupStatusFailed, Error: "validation failed"}))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/", test.write)
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			router.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
			}
			var response map[string]json.RawMessage
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
			}
			got, ok := response[test.wantKey]
			if !ok || string(got) != test.wantJSON {
				t.Fatalf("response = %s, want %s=%s", recorder.Body.String(), test.wantKey, test.wantJSON)
			}
			if test.name == "backup status" && strings.Contains(recorder.Body.String(), "fileContents") {
				t.Fatalf("backup response contains file contents: %s", recorder.Body.String())
			}
		})
	}
}

func mustMarshalJSON(t *testing.T, value any) string {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	return string(payload)
}
