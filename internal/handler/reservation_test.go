package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"rented/internal/db"
)

const testPropertyID = "550e8400-e29b-41d4-a716-446655440000"

func TestHandler_ListReservationsRejectsBadFilterDate(t *testing.T) {
	h := &Handler{}
	session := &SessionData{UserID: "42", UserType: "manager"}

	rec := runJSONHandlerWithSession(
		t,
		http.MethodPost,
		"/reservation",
		`{"check_in_from":"not-a-date"}`,
		session,
		h.ListReservations,
	)

	assertErrorResponse(t, rec, http.StatusBadRequest, "invalid check_in_from, expected RFC3339")
}

func TestHandler_ListReservationsSuccess(t *testing.T) {
	queries := &fakeQueries{}
	h := &Handler{queries: queries}
	session := &SessionData{UserID: "42", UserType: "manager"}
	body := `{"page":2,` +
		`"property_name":"Cabin",` +
		`"guest_name":"Guest",` +
		`"check_in_from":"2026-07-01T00:00:00Z",` +
		`"check_out_to":"2026-07-10T00:00:00Z"}`

	rec := runJSONHandlerWithSession(
		t,
		http.MethodPost,
		"/reservation",
		body,
		session,
		h.ListReservations,
	)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	params := queries.listReservationsFilteredParam
	if params.OwnerID != 42 || params.Off != pageSize || params.Lim != pageSize {
		t.Fatalf("params = %#v, want owner 42, offset %d, limit %d", params, pageSize, pageSize)
	}
	if !params.PropertyNameFilter.Valid || params.PropertyNameFilter.String != "Cabin" {
		t.Fatalf("property filter = %#v, want Cabin", params.PropertyNameFilter)
	}
	if !params.GuestNameFilter.Valid || params.GuestNameFilter.String != "Guest" {
		t.Fatalf("guest filter = %#v, want Guest", params.GuestNameFilter)
	}
	if !params.CheckInFrom.Valid || !params.CheckOutTo.Valid {
		t.Fatalf(
			"date filters valid = (%t, %t), want both true",
			params.CheckInFrom.Valid,
			params.CheckOutTo.Valid,
		)
	}
}

func TestHandler_CreateReservationRuleConflict(t *testing.T) {
	rules := &fakeRules{errs: []error{errors.New("overlap")}}
	h := &Handler{rules: rules}
	session := &SessionData{UserID: "42", UserType: "guest"}

	rec := runJSONHandlerWithSession(
		t,
		http.MethodPost,
		"/reservation/new",
		reservationBody(),
		session,
		h.CreateReservation,
	)

	assertErrorResponse(t, rec, http.StatusConflict, "overlap")
}

func TestHandler_CreateReservationSuccess(t *testing.T) {
	queries := &fakeQueries{
		getPropertyByIDFunc: func(_ context.Context, _ pgtype.UUID) (db.Property, error) {
			return db.Property{OwnerID: 42, Title: "Cabin"}, nil
		},
		createReservationFunc: func(
			_ context.Context,
			_ db.CreateReservationParams,
		) (db.Reservation, error) {
			return db.Reservation{ID: 7}, nil
		},
	}
	h := &Handler{queries: queries, rules: &fakeRules{}}
	session := &SessionData{UserID: "42", UserType: "guest"}

	rec := runJSONHandlerWithSession(
		t,
		http.MethodPost,
		"/reservation/new",
		reservationBody(),
		session,
		h.CreateReservation,
	)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	params := queries.createReservationParams
	if params.BookedBy != 42 || params.PropertyName != "Cabin" || params.GuestName != "Test Guest" {
		t.Fatalf("params = %#v, want booked_by 42, property Cabin, guest Test Guest", params)
	}
	if !params.CheckIn.Valid || !params.CheckOut.Valid {
		t.Fatalf(
			"reservation timestamps valid = (%t, %t), want both true",
			params.CheckIn.Valid,
			params.CheckOut.Valid,
		)
	}
}

func reservationBody() string {
	return `{"property_id":"` + testPropertyID + `",` +
		`"guest_name":"Test Guest",` +
		`"checkin":"2026-07-01",` +
		`"checkout":"2026-07-05",` +
		`"timezone":"UTC"}`
}
