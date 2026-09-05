package service

import "testing"

func TestValidateDatabaseImportSizeEnforcesMaximum(t *testing.T) {
	if err := validateDatabaseImportSize(maxDatabaseImportBytes); err != nil {
		t.Fatalf("maximum allowed size rejected: %v", err)
	}
	if err := validateDatabaseImportSize(maxDatabaseImportBytes + 1); err == nil {
		t.Fatal("database import larger than maximum was accepted")
	}
	if err := validateDatabaseImportSize(-1); err == nil {
		t.Fatal("negative database import size was accepted")
	}
}
