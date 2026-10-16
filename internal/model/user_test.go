package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestUserPasswordHashIsNotSerialized(t *testing.T) {
	user := User{ID: 1, Email: "reader@example.com", PasswordHash: "private-hash", Role: RoleViewer}
	encoded, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if strings.Contains(string(encoded), "password_hash") || strings.Contains(string(encoded), "private-hash") {
		t.Fatalf("serialized user exposes password hash: %s", encoded)
	}
}
