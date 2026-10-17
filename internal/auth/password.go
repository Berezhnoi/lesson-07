package auth

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type PasswordValidationError struct {
	Message string
}

func (validationError PasswordValidationError) Error() string {
	return validationError.Message
}

func HashPassword(plain string) (string, error) {
	if len(plain) > 72 {
		return "", PasswordValidationError{Message: "password must be at most 72 bytes"}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
