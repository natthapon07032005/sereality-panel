package service

import (
	"encoding/json"
	"strings"
	"testing"

	"x-ui/database/model"
)

func TestPasswordHashCanBeVerifiedWithoutStoringPlaintext(t *testing.T) {
	password := "correct horse battery staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	if hash == password {
		t.Fatal("password hash must not equal plaintext")
	}
	if !CheckPasswordHash(password, hash) {
		t.Fatal("expected password hash to verify")
	}
	if CheckPasswordHash("wrong password", hash) {
		t.Fatal("wrong password must not verify")
	}
}

func TestToUserSecretViewOmitsPasswordHash(t *testing.T) {
	view := ToUserSecretView(model.User{Id: 8, Username: "admin", Password: "bcrypt-hash", LoginSecret: "panel-secret"})
	if view.ID != 8 || view.Username != "admin" || view.LoginSecret != "panel-secret" {
		t.Fatalf("user secret view = %+v", view)
	}
	payload, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal user secret view: %v", err)
	}
	if strings.Contains(string(payload), "bcrypt-hash") || strings.Contains(string(payload), "password") {
		t.Fatalf("user secret view exposed password data: %s", payload)
	}
}

func TestPasswordMatchesStoredHashOrLegacyValue(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	if !passwordMatchesStored(hash, "correct-password") {
		t.Fatal("bcrypt password should verify")
	}
	if passwordMatchesStored(hash, "wrong-password") {
		t.Fatal("wrong bcrypt password should not verify")
	}
	if !passwordMatchesStored("legacy-password", "legacy-password") {
		t.Fatal("legacy plaintext password should verify for migration")
	}
}
