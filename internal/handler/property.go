package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"rented/internal/db"
)

const pageSize = 50

type createPropertyRequest struct {
	Title   string `json:"title"`
	Address string `json:"address"`
}

type propertyResponse struct {
	ID              pgtype.UUID        `json:"id"`
	OwnerID         int64              `json:"owner_id"`
	Title           string             `json:"title"`
	Address         string             `json:"address"`
	CreatedAt       pgtype.Timestamptz `json:"created_at"`
	UpdatedAt       pgtype.Timestamptz `json:"updated_at"`
	Status          string             `json:"status"`
	GuestName       *string            `json:"guest_name,omitempty"`
	CurrentCheckIn  *time.Time         `json:"current_check_in,omitempty"`
	CurrentCheckOut *time.Time         `json:"current_check_out,omitempty"`
	NextCheckIn     *time.Time         `json:"next_check_in,omitempty"`
}

type propertyReservationStatus struct {
	status          string
	guestName       *string
	currentCheckIn  *time.Time
	currentCheckOut *time.Time
	nextCheckIn     *time.Time
}

type listPropertiesResponse struct {
	Properties []propertyResponse `json:"properties"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	Pages      int                `json:"pages"`
}

func (h *Handler) ListProperties(c echo.Context) error {
	session, ok := c.Get("session").(*SessionData)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, ErrorCodeUnauthorized, "unauthorized")
	}

	var req struct {
		Page int `json:"page" query:"page"`
	}
	if err := c.Bind(&req); err != nil {
		h.logger.Error("list properties: failed to bind request", err)
		return invalidRequestBody(c)
	}

	if req.Page < 1 {
		req.Page = 1
	}
	offset := (req.Page - 1) * pageSize

	ownerID, err := strconv.ParseInt(session.UserID, 10, 64)
	if err != nil {
		h.logger.Error("list properties: invalid session user id", err)
		return errorResponse(c, http.StatusUnauthorized, ErrorCodeUnauthorized, "unauthorized")
	}

	properties, err := h.queries.ListPropertiesByOwner(c.Request().Context(), db.ListPropertiesByOwnerParams{
		OwnerID: ownerID,
		Limit:   int32(pageSize),
		Offset:  int32(offset),
	})
	if err != nil {
		return h.internalError(c, "list properties: db query failed", err)
	}

	var total int64
	if len(properties) > 0 {
		total = properties[0].TotalCount
	}

	statuses := map[pgtype.UUID]propertyReservationStatus{}
	if len(properties) > 0 {
		propertyIDs := make([]pgtype.UUID, 0, len(properties))
		for _, property := range properties {
			propertyIDs = append(propertyIDs, property.ID)
		}

		reservations, err := h.queries.ListActiveOrUpcomingReservationsByPropertyIDs(c.Request().Context(), propertyIDs)
		if err != nil {
			return h.internalError(c, "list properties: reservation status query failed", err)
		}

		now := time.Now()
		for _, reservation := range reservations {
			status := statuses[reservation.PropertyID]
			if reservation.CheckIn.Time.After(now) {
				if status.nextCheckIn == nil {
					nextCheckIn := reservation.CheckIn.Time
					status.nextCheckIn = &nextCheckIn
					if status.status != "occupied" {
						guestName := reservation.GuestName
						status.guestName = &guestName
					}
				}
			} else if status.status != "occupied" {
				guestName := reservation.GuestName
				currentCheckIn := reservation.CheckIn.Time
				currentCheckOut := reservation.CheckOut.Time
				status.status = "occupied"
				status.guestName = &guestName
				status.currentCheckIn = &currentCheckIn
				status.currentCheckOut = &currentCheckOut
			}
			statuses[reservation.PropertyID] = status
		}
	}

	pages := calcPages(int(total), pageSize)
	items := make([]propertyResponse, 0, len(properties))
	for _, property := range properties {
		status := statuses[property.ID]
		if status.status == "" {
			status.status = "vacant"
		}

		items = append(items, propertyResponse{
			ID:              property.ID,
			OwnerID:         property.OwnerID,
			Title:           property.Title,
			Address:         property.Address,
			CreatedAt:       property.CreatedAt,
			UpdatedAt:       property.UpdatedAt,
			Status:          status.status,
			GuestName:       status.guestName,
			CurrentCheckIn:  status.currentCheckIn,
			CurrentCheckOut: status.currentCheckOut,
			NextCheckIn:     status.nextCheckIn,
		})
	}

	return c.JSON(http.StatusOK, listPropertiesResponse{
		Properties: items,
		Total:      total,
		Page:       req.Page,
		Pages:      pages,
	})
}

func (h *Handler) CreateProperty(c echo.Context) error {
	session, ok := c.Get("session").(*SessionData)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, ErrorCodeUnauthorized, "unauthorized")
	}

	if session.UserType != "manager" {
		return errorResponse(c, http.StatusForbidden, ErrorCodeForbidden, "only managers can create properties")
	}

	var req createPropertyRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("create property: failed to bind request", err)
		return invalidRequestBody(c)
	}

	if req.Title == "" || req.Address == "" {
		return errorResponse(c, http.StatusBadRequest, ErrorCodeMissingRequiredFields, "title and address are required")
	}

	ownerID, err := strconv.ParseInt(session.UserID, 10, 64)
	if err != nil {
		h.logger.Error("create property: invalid session user id", err)
		return errorResponse(c, http.StatusUnauthorized, ErrorCodeUnauthorized, "unauthorized")
	}

	property, err := h.queries.CreateProperty(c.Request().Context(), db.CreatePropertyParams{
		OwnerID: ownerID,
		Title:   req.Title,
		Address: req.Address,
	})
	if err != nil {
		return h.internalError(c, "create property: db query failed", err)
	}

	return c.JSON(http.StatusCreated, property)
}
