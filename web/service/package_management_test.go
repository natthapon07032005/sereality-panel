package service

import "testing"

func TestValidatePackageInputRejectsInvalidLimits(t *testing.T) {
	invalid := []PackageInput{
		{Name: "", DurationDays: 30},
		{Name: "Basic", DurationDays: 0},
		{Name: "Basic", DurationDays: 30, TrafficLimit: -1},
		{Name: "Basic", DurationDays: 30, DeviceLimit: -1},
	}
	for _, input := range invalid {
		if err := ValidatePackageInput(input); err == nil {
			t.Fatalf("ValidatePackageInput(%+v) accepted invalid input", input)
		}
	}
}

func TestValidatePackageInputAcceptsUnlimitedTrafficAndDevices(t *testing.T) {
	input := PackageInput{Name: "Unlimited", DurationDays: 30, TrafficLimit: 0, DeviceLimit: 0}
	if err := ValidatePackageInput(input); err != nil {
		t.Fatalf("ValidatePackageInput() error = %v", err)
	}
}

func TestPackageDeletionIsBlockedWhenReferenced(t *testing.T) {
	if PackageDeletionAllowed(0, 1) {
		t.Fatal("package with subscription references was allowed to delete")
	}
	if PackageDeletionAllowed(1, 0) {
		t.Fatal("package with user references was allowed to delete")
	}
	if !PackageDeletionAllowed(0, 0) {
		t.Fatal("unreferenced package was not allowed to delete")
	}
}
