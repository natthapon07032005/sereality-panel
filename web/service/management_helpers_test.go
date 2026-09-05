package service

import "testing"

func TestRequireRowsAffectedRejectsMissingManagementResource(t *testing.T) {
	if err := requireRowsAffected("package", 0); err == nil || err.Error() != "package not found" {
		t.Fatalf("missing package error = %v, want package not found", err)
	}
	if err := requireRowsAffected("node", 1); err != nil {
		t.Fatalf("updated node returned error: %v", err)
	}
}
