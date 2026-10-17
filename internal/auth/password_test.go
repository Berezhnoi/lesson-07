package auth

import (
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHashing(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "hash differs and verifies",
			run: func(t *testing.T) {
				first, err := HashPassword("correct horse battery")
				if err != nil {
					t.Fatalf("HashPassword(first) error = %v", err)
				}
				second, err := HashPassword("correct horse battery")
				if err != nil {
					t.Fatalf("HashPassword(second) error = %v", err)
				}
				if first == second {
					t.Fatal("separate password hashes are identical")
				}
				if err := CheckPassword(first, "correct horse battery"); err != nil {
					t.Fatalf("CheckPassword() error = %v", err)
				}
				if err := CheckPassword(first, "incorrect password"); !errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
					t.Fatalf("CheckPassword(wrong password) error = %v, want bcrypt mismatch", err)
				}
			},
		},
		{
			name: "rejects passwords longer than bcrypt limit",
			run: func(t *testing.T) {
				_, err := HashPassword(string(make([]byte, 73)))
				var validationError PasswordValidationError
				if !errors.As(err, &validationError) {
					t.Fatalf("HashPassword() error = %v, want PasswordValidationError", err)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}
