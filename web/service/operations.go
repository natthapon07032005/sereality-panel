package service

import (
	"sync"
	"time"
)

// RateLimitDecision is the result of evaluating one attempt against a
// sliding time window. RetryAfter is zero when the attempt is allowed.
type RateLimitDecision struct {
	Allowed    bool          `json:"allowed"`
	Limit      int           `json:"limit"`
	Remaining  int           `json:"remaining"`
	RetryAfter time.Duration `json:"retryAfter"`
}

// DecideRateLimit evaluates an attempt without changing the supplied history.
// Attempts at exactly now-window are expired; newer attempts remain active.
func DecideRateLimit(attempts []time.Time, now time.Time, limit int, window time.Duration) RateLimitDecision {
	if limit <= 0 || window <= 0 {
		return RateLimitDecision{Limit: limit}
	}

	cutoff := now.Add(-window)
	active := 0
	var oldest time.Time
	for _, attempt := range attempts {
		if !attempt.After(cutoff) {
			continue
		}
		active++
		if oldest.IsZero() || attempt.Before(oldest) {
			oldest = attempt
		}
	}

	if active >= limit {
		retryAfter := oldest.Add(window).Sub(now)
		if retryAfter < 0 {
			retryAfter = 0
		}
		return RateLimitDecision{
			Allowed:    false,
			Limit:      limit,
			RetryAfter: retryAfter,
		}
	}

	return RateLimitDecision{
		Allowed:   true,
		Limit:     limit,
		Remaining: limit - active - 1,
	}
}

type rateLimitBucket struct {
	attempts []time.Time
	sequence uint64
}

// RateLimiter is a bounded, in-memory sliding-window limiter. The explicit
// AllowAt method keeps the stateful wrapper deterministic in tests while Allow
// provides the normal wall-clock API for callers.
type RateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	maxKeys  int
	sequence uint64
	buckets  map[string]rateLimitBucket
}

// NewRateLimiter creates a limiter with at most maxKeys tracked keys. Invalid
// limits fail closed when an attempt is evaluated rather than panicking.
func NewRateLimiter(limit int, window time.Duration, maxKeys int) *RateLimiter {
	return &RateLimiter{
		limit:   limit,
		window:  window,
		maxKeys: maxKeys,
		buckets: make(map[string]rateLimitBucket),
	}
}

// Allow evaluates and records an attempt using the current wall-clock time.
func (r *RateLimiter) Allow(key string) RateLimitDecision {
	return r.AllowAt(key, time.Now())
}

// AllowAt evaluates and records an attempt at now. Accepted attempts are
// retained only for the configured window, and denied attempts do not extend
// the window.
func (r *RateLimiter) AllowAt(key string, now time.Time) RateLimitDecision {
	if r == nil {
		return RateLimitDecision{}
	}
	if r.limit <= 0 || r.window <= 0 || r.maxKeys <= 0 {
		return RateLimitDecision{Limit: r.limit}
	}

	key = normalizeRateLimitKey(key)
	r.mu.Lock()
	defer r.mu.Unlock()

	r.pruneExpired(now)
	bucket, exists := r.buckets[key]
	if !exists {
		if len(r.buckets) >= r.maxKeys {
			r.evictOldest()
		}
		r.sequence++
		bucket = rateLimitBucket{sequence: r.sequence}
	}

	cutoff := now.Add(-r.window)
	activeAttempts := bucket.attempts[:0]
	for _, attempt := range bucket.attempts {
		if attempt.After(cutoff) {
			activeAttempts = append(activeAttempts, attempt)
		}
	}
	bucket.attempts = activeAttempts

	decision := DecideRateLimit(bucket.attempts, now, r.limit, r.window)
	if decision.Allowed {
		bucket.attempts = append(bucket.attempts, now)
	}
	r.sequence++
	bucket.sequence = r.sequence
	r.buckets[key] = bucket
	return decision
}

// TrackedKeyCount reports the current number of in-memory key buckets.
func (r *RateLimiter) TrackedKeyCount() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.buckets)
}

func (r *RateLimiter) pruneExpired(now time.Time) {
	cutoff := now.Add(-r.window)
	for key, bucket := range r.buckets {
		active := bucket.attempts[:0]
		for _, attempt := range bucket.attempts {
			if attempt.After(cutoff) {
				active = append(active, attempt)
			}
		}
		if len(active) == 0 {
			delete(r.buckets, key)
			continue
		}
		bucket.attempts = active
		r.buckets[key] = bucket
	}
}

func (r *RateLimiter) evictOldest() {
	var oldestKey string
	var oldestSequence uint64
	for key, bucket := range r.buckets {
		if oldestKey == "" || bucket.sequence < oldestSequence {
			oldestKey = key
			oldestSequence = bucket.sequence
		}
	}
	if oldestKey != "" {
		delete(r.buckets, oldestKey)
	}
}

func normalizeRateLimitKey(key string) string {
	for len(key) > 0 && (key[0] == ' ' || key[0] == '\t' || key[0] == '\n' || key[0] == '\r') {
		key = key[1:]
	}
	for len(key) > 0 {
		last := key[len(key)-1]
		if last != ' ' && last != '\t' && last != '\n' && last != '\r' {
			break
		}
		key = key[:len(key)-1]
	}
	if key == "" {
		return "<unknown>"
	}
	return key
}

// AuditEventInput contains only fields that are safe to copy into an audit
// event. Credentials, tokens, and arbitrary metadata are intentionally absent.
type AuditEventInput struct {
	EventType string    `json:"eventType,omitempty"`
	Action    string    `json:"action"`
	Username  string    `json:"username,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	RemoteIP  string    `json:"remoteIp,omitempty"`
	RequestID string    `json:"requestId,omitempty"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// AuditEvent is the safe, serializable audit representation. It deliberately
// has no password, secret, token, request-body, or arbitrary-details fields.
type AuditEvent struct {
	EventType string    `json:"eventType,omitempty"`
	Action    string    `json:"action"`
	Username  string    `json:"username,omitempty"`
	Resource  string    `json:"resource,omitempty"`
	RemoteIP  string    `json:"remoteIp,omitempty"`
	RequestID string    `json:"requestId,omitempty"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// NewAuditEvent constructs an audit DTO from its allowlisted fields.
func NewAuditEvent(input AuditEventInput) AuditEvent {
	return AuditEvent{
		EventType: input.EventType,
		Action:    input.Action,
		Username:  input.Username,
		Resource:  input.Resource,
		RemoteIP:  input.RemoteIP,
		RequestID: input.RequestID,
		Success:   input.Success,
		Timestamp: input.Timestamp,
	}
}

const (
	BackupStatusPending   = "pending"
	BackupStatusRunning   = "running"
	BackupStatusSucceeded = "succeeded"
	BackupStatusFailed    = "failed"
)

// BackupStatusInput is metadata about a backup operation. It does not accept
// a file or byte payload.
type BackupStatusInput struct {
	SourcePath      string    `json:"sourcePath,omitempty"`
	DestinationPath string    `json:"destinationPath,omitempty"`
	Status          string    `json:"status"`
	Error           string    `json:"error,omitempty"`
	StartedAt       time.Time `json:"startedAt,omitempty"`
	CompletedAt     time.Time `json:"completedAt,omitempty"`
}

// BackupStatus is a safe operational DTO that reports backup metadata only.
type BackupStatus struct {
	SourcePath      string    `json:"sourcePath,omitempty"`
	DestinationPath string    `json:"destinationPath,omitempty"`
	Status          string    `json:"status"`
	Error           string    `json:"error,omitempty"`
	StartedAt       time.Time `json:"startedAt,omitempty"`
	CompletedAt     time.Time `json:"completedAt,omitempty"`
}

// BuildBackupStatus constructs a status DTO without reading or embedding any
// file contents.
func BuildBackupStatus(input BackupStatusInput) BackupStatus {
	return BackupStatus{
		SourcePath:      input.SourcePath,
		DestinationPath: input.DestinationPath,
		Status:          input.Status,
		Error:           input.Error,
		StartedAt:       input.StartedAt,
		CompletedAt:     input.CompletedAt,
	}
}
