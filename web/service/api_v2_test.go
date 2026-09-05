package service

import (
	"errors"
	"strings"
	"testing"
)

func TestNewAPIV2ResponseMarksErrorsWithoutLeakingData(t *testing.T) {
	response := NewAPIV2Response(map[string]string{"token": "do-not-return"}, errors.New("database failed: token=do-not-return"))
	if response.Success || response.Data != nil || response.Error == nil {
		t.Fatalf("error response = %+v", response)
	}
	if strings.Contains(response.Error.Message, "do-not-return") {
		t.Fatalf("error response leaked internal detail: %q", response.Error.Message)
	}
}

func TestBuildNodeSyncResultCountsDeterministicChanges(t *testing.T) {
	result := BuildNodeSyncResult(4, []string{"a", "b"}, []string{"b", "c"})
	if result.NodeID != 4 || result.Added != 1 || result.Removed != 1 || result.Updated != 1 {
		t.Fatalf("sync result = %+v", result)
	}
}
