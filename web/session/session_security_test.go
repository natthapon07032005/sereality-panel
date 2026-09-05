package session

import "testing"

func TestIsActiveUserStatusRejectsSuspendedAndExpired(t *testing.T) {
	for _, status := range []string{"suspended", "expired"} {
		if isActiveUserStatus(status) {
			t.Fatalf("status %q must not be treated as active", status)
		}
	}
	for _, status := range []string{"", "active"} {
		if !isActiveUserStatus(status) {
			t.Fatalf("status %q must be treated as active for legacy/active users", status)
		}
	}
}
