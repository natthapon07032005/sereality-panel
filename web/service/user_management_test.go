package service

import (
	"testing"

	"x-ui/database/model"
)

func TestValidateManagedUserStatusRejectsUnknownValues(t *testing.T) {
	if err := ValidateManagedUserStatus("paused"); err == nil {
		t.Fatal("unknown user status was accepted")
	}
	for _, status := range []string{ManagedUserStatusActive, ManagedUserStatusSuspended, ManagedUserStatusExpired} {
		if err := ValidateManagedUserStatus(status); err != nil {
			t.Fatalf("status %q rejected: %v", status, err)
		}
	}
}

func TestToManagedUserNeverExposesCredentials(t *testing.T) {
	user := model.User{Id: 7, Username: "admin", Password: "bcrypt-hash", LoginSecret: "secret-token"}
	managed := ToManagedUser(user)
	if managed.ID != 7 || managed.Username != "admin" {
		t.Fatalf("managed user identity = %+v", managed)
	}
	if managed.Password != "" || managed.LoginSecret != "" {
		t.Fatalf("managed user exposed credentials: %+v", managed)
	}
}
