package utils

import (
	"errors"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

var ErrInvalidEmail = errors.New("invalid email format")
var ErrWeakPassword = errors.New("password must be at least 8 characters")

func ValidateEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return ErrInvalidEmail
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}
	return nil
}

func SanitizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
