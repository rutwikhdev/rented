package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func TestHandler_Health(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	h := &Handler{}
	if err := h.Health(c); err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status body = %q, want %q", body["status"], "ok")
	}
}

func TestErrorResponse(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := errorResponse(c, http.StatusBadRequest, "bad request"); err != nil {
		t.Fatalf("errorResponse() error = %v", err)
	}

	assertErrorResponse(t, rec, http.StatusBadRequest, "bad request")
}

func TestInvalidRequestBody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	if err := invalidRequestBody(c); err != nil {
		t.Fatalf("invalidRequestBody() error = %v", err)
	}

	assertErrorResponse(t, rec, http.StatusBadRequest, "invalid request body")
}

func TestNormalizePage(t *testing.T) {
	tests := []struct {
		name       string
		page       int
		wantPage   int
		wantOffset int32
	}{
		{
			name:       "default page",
			page:       0,
			wantPage:   1,
			wantOffset: 0,
		},
		{
			name:       "second page",
			page:       2,
			wantPage:   2,
			wantOffset: pageSize,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPage, gotOffset := normalizePage(tt.page)
			if gotPage != tt.wantPage || gotOffset != tt.wantOffset {
				t.Fatalf("normalizePage(%d) = (%d, %d), want (%d, %d)",
					tt.page,
					gotPage,
					gotOffset,
					tt.wantPage,
					tt.wantOffset,
				)
			}
		})
	}
}

func TestPGTimestamp(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	got := pgTimestamp(now)
	if !got.Valid {
		t.Fatal("pgTimestamp().Valid = false, want true")
	}
	if !got.Time.Equal(now) {
		t.Fatalf("pgTimestamp().Time = %v, want %v", got.Time, now)
	}
}

func TestTextFilter(t *testing.T) {
	empty := textFilter("")
	if empty.Valid {
		t.Fatal("textFilter(empty).Valid = true, want false")
	}

	filled := textFilter("guest")
	if !filled.Valid || filled.String != "guest" {
		t.Fatalf("textFilter() = (%q, %t), want (%q, true)", filled.String, filled.Valid, "guest")
	}
}

func assertErrorResponse(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantError string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d", rec.Code, wantStatus)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != wantError {
		t.Fatalf("error body = %q, want %q", body["error"], wantError)
	}
}
