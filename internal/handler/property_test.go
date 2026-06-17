package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"rented/internal/db"
)

func TestHandler_ListPropertiesSuccess(t *testing.T) {
	queries := &fakeQueries{
		listPropertiesByOwnerFunc: func(
			_ context.Context,
			_ db.ListPropertiesByOwnerParams,
		) ([]db.ListPropertiesByOwnerRow, error) {
			return []db.ListPropertiesByOwnerRow{{TotalCount: 75}}, nil
		},
	}
	h := &Handler{queries: queries}
	session := &SessionData{UserID: "42", UserType: "manager"}

	rec := runJSONHandlerWithSession(t, http.MethodPost, "/property", `{}`, session, h.ListProperties)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if queries.listPropertiesByOwnerParam.OwnerID != 42 {
		t.Fatalf("owner id = %d, want %d", queries.listPropertiesByOwnerParam.OwnerID, 42)
	}
	if queries.listPropertiesByOwnerParam.Offset != 0 {
		t.Fatalf("offset = %d, want 0", queries.listPropertiesByOwnerParam.Offset)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["page"] != float64(1) || body["pages"] != float64(2) || body["total"] != float64(75) {
		t.Fatalf("response pagination = %#v, want page 1, pages 2, total 75", body)
	}
}

func TestHandler_CreatePropertyRequiresManager(t *testing.T) {
	h := &Handler{}
	session := &SessionData{UserID: "42", UserType: "guest"}

	rec := runJSONHandlerWithSession(
		t,
		http.MethodPost,
		"/property/new",
		`{"title":"Cabin","address":"1 Main St"}`,
		session,
		h.CreateProperty,
	)

	assertErrorResponse(t, rec, http.StatusForbidden, "only managers can create properties")
}

func TestHandler_CreatePropertySuccess(t *testing.T) {
	queries := &fakeQueries{}
	h := &Handler{queries: queries}
	session := &SessionData{UserID: "42", UserType: "manager"}

	rec := runJSONHandlerWithSession(
		t,
		http.MethodPost,
		"/property/new",
		`{"title":"Cabin","address":"1 Main St"}`,
		session,
		h.CreateProperty,
	)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if queries.createPropertyParams.OwnerID != 42 {
		t.Fatalf("owner id = %d, want %d", queries.createPropertyParams.OwnerID, 42)
	}
	if queries.createPropertyParams.Title != "Cabin" {
		t.Fatalf("title = %q, want %q", queries.createPropertyParams.Title, "Cabin")
	}
}
