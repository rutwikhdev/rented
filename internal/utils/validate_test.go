package utils

import (
	"errors"
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:  "valid email",
			email: "test@example.com",
		},
		{
			name:  "mixed case valid email",
			email: "Test.User@example.COM",
		},
		{
			name:    "missing at sign",
			email:   "test.example.com",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "missing domain suffix",
			email:   "test@example",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: ErrInvalidEmail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateEmail() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "eight characters",
			password: "12345678",
		},
		{
			name:     "longer password",
			password: "longer-password",
		},
		{
			name:     "seven characters",
			password: "1234567",
			wantErr:  ErrWeakPassword,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  ErrWeakPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidatePassword() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{
			name:  "trims whitespace",
			email: " test@example.com ",
			want:  "test@example.com",
		},
		{
			name:  "lowercases",
			email: "Test@Example.COM",
			want:  "test@example.com",
		},
		{
			name:  "trims and lowercases",
			email: "  Test.User@Example.COM  ",
			want:  "test.user@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeEmail(tt.email); got != tt.want {
				t.Fatalf("SanitizeEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}
