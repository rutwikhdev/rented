package handler

import (
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"rented/internal/db"
	"rented/internal/rules"
	"rented/internal/utils"
)

type listReservationsRequest struct {
	Page         int    `json:"page"`
	PropertyName string `json:"property_name"`
	GuestName    string `json:"guest_name"`
	CheckInFrom  string `json:"check_in_from"`
	CheckOutTo   string `json:"check_out_to"`
}

func (h *Handler) ListReservations(c echo.Context) error {
	session, ok := c.Get("session").(*SessionData)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, "unauthorized")
	}

	var req listReservationsRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("list reservations: failed to bind request", err)
		return invalidRequestBody(c)
	}

	if req.Page < 1 {
		req.Page = 1
	}
	offset := int32((req.Page - 1) * pageSize)

	userID, err := strconv.ParseInt(session.UserID, 10, 64)
	if err != nil {
		h.logger.Error("list reservations: invalid session user id", err)
		return errorResponse(c, http.StatusUnauthorized, "unauthorized")
	}

	params := db.ListManagerReservationsFilteredParams{
		OwnerID: userID,
		Off:     offset,
		Lim:     int32(pageSize),
	}

	if req.PropertyName != "" {
		params.PropertyNameFilter = pgtype.Text{String: req.PropertyName, Valid: true}
	}
	if req.GuestName != "" {
		params.GuestNameFilter = pgtype.Text{String: req.GuestName, Valid: true}
	}
	if req.CheckInFrom != "" {
		t, err := time.Parse(time.RFC3339, req.CheckInFrom)
		if err != nil {
			return errorResponse(c, http.StatusBadRequest, "invalid check_in_from, expected RFC3339")
		}
		params.CheckInFrom = pgtype.Timestamptz{Time: t, Valid: true}
	}
	if req.CheckOutTo != "" {
		t, err := time.Parse(time.RFC3339, req.CheckOutTo)
		if err != nil {
			return errorResponse(c, http.StatusBadRequest, "invalid check_out_to, expected RFC3339")
		}
		params.CheckOutTo = pgtype.Timestamptz{Time: t, Valid: true}
	}

	rows, err := h.queries.ListManagerReservationsFiltered(c.Request().Context(), params)
	if err != nil {
		return h.internalError(c, "list reservations: db query failed", err)
	}
	if rows == nil {
		rows = []db.ListManagerReservationsFilteredRow{}
	}

	var total int64
	if len(rows) > 0 {
		total = rows[0].TotalCount
	}

	pages := calcPages(int(total), pageSize)

	return c.JSON(http.StatusOK, map[string]any{
		"reservations": rows,
		"total":        total,
		"page":         req.Page,
		"pages":        pages,
	})
}

type createReservationRequest struct {
	PropertyID string `json:"property_id"`
	GuestName  string `json:"guest_name"`
	CheckIn    string `json:"checkin"`
	CheckOut   string `json:"checkout"`
	Timezone   string `json:"timezone"`
}

func (h *Handler) lockReservationProperty(propertyID string) func() {
	actual, _ := h.reservationLocks.LoadOrStore(propertyID, &sync.Mutex{})
	mu := actual.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

func (h *Handler) CreateReservation(c echo.Context) error {
	session, ok := c.Get("session").(*SessionData)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, "unauthorized")
	}

	var req createReservationRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("create reservation: failed to bind request", err)
		return invalidRequestBody(c)
	}

	if req.PropertyID == "" || req.GuestName == "" || req.CheckIn == "" || req.CheckOut == "" {
		return errorResponse(c, http.StatusBadRequest, "property_id, guest_name, checkin, and checkout are required")
	}
	if req.Timezone == "" {
		return errorResponse(c, http.StatusBadRequest, "timezone is required")
	}

	var pid pgtype.UUID
	if err := pid.Scan(req.PropertyID); err != nil {
		return errorResponse(c, http.StatusBadRequest, "invalid property_id")
	}

	property, err := h.queries.GetPropertyByID(c.Request().Context(), pid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errorResponse(c, http.StatusNotFound, "property not found")
		}
		return h.internalError(c, "create reservation: failed to get property", err)
	}

	if strconv.FormatInt(property.OwnerID, 10) != session.UserID && session.UserType != "manager" {
		return errorResponse(c, http.StatusNotFound, "property not found")
	}

	checkIn, err := utils.ParseLocalToUTC(req.CheckIn, req.Timezone, true)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "invalid checkin datetime or timezone")
	}
	checkOut, err := utils.ParseLocalToUTC(req.CheckOut, req.Timezone, false)
	if err != nil {
		return errorResponse(c, http.StatusBadRequest, "invalid checkout datetime or timezone")
	}

	bookedBy, err := strconv.ParseInt(session.UserID, 10, 64)
	if err != nil {
		h.logger.Error("create reservation: invalid session user id", err)
		return errorResponse(c, http.StatusUnauthorized, "unauthorized")
	}

	var checkInTs pgtype.Timestamptz
	checkInTs.Time = checkIn
	checkInTs.Valid = true

	var checkOutTs pgtype.Timestamptz
	checkOutTs.Time = checkOut
	checkOutTs.Valid = true

	unlock := h.lockReservationProperty(req.PropertyID)
	defer unlock()

	if errs := h.rules.Run(c.Request().Context(), rules.RuleInput{
		PropertyID: pid,
		CheckIn:    checkIn,
		CheckOut:   checkOut,
	}); len(errs) > 0 {
		return errorResponse(c, http.StatusConflict, errs[0].Error())
	}

	reservation, err := h.queries.CreateReservation(c.Request().Context(), db.CreateReservationParams{
		PropertyID:   pid,
		PropertyName: property.Title,
		BookedBy:     bookedBy,
		GuestName:    req.GuestName,
		CheckIn:      checkInTs,
		CheckOut:     checkOutTs,
	})
	if err != nil {
		return h.internalError(c, "create reservation: db query failed", err)
	}

	return c.JSON(http.StatusCreated, reservation)
}
