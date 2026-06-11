package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestHandler_SignupValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "missing required fields",
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "name, email, and password are required",
		},
		{
			name:       "invalid email",
			body:       `{"name":"Test User","email":"invalid","password":"password123"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid email format",
		},
		{
			name:       "weak password",
			body:       `{"name":"Test User","email":"test@example.com","password":"short"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "password must be at least 8 characters",
		},
		{
			name:       "invalid user type",
			body:       `{"name":"Test User","email":"test@example.com","password":"password123","type":"admin"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "type must be guest or manager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := runJSONHandler(t, http.MethodPost, "/signup", tt.body, (&Handler{}).Signup)
			assertErrorResponse(t, rec, tt.wantStatus, tt.wantError)
		})
	}
}

func TestHandler_LoginValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantError  string
	}{
		{
			name:       "missing email",
			body:       `{"password":"password123"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "email and password are required",
		},
		{
			name:       "missing password",
			body:       `{"email":"test@example.com"}`,
			wantStatus: http.StatusBadRequest,
			wantError:  "email and password are required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := runJSONHandler(t, http.MethodPost, "/login", tt.body, (&Handler{}).Login)
			assertErrorResponse(t, rec, tt.wantStatus, tt.wantError)
		})
	}
}

func runJSONHandler(
	t *testing.T,
	method string,
	target string,
	body string,
	handler func(echo.Context) error,
) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := handler(c); err != nil {
		t.Fatalf("handler error = %v", err)
	}

	return rec
}
