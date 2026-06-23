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

	rows, err := h.queries.ListPropertiesWithOccupantByOwner(c.Request().Context(), db.ListPropertiesWithOccupantByOwnerParams{
		OwnerID: ownerID,
		Limit:   int32(pageSize),
		Offset:  int32(offset),
	})
	if err != nil {
		return h.internalError(c, "list properties: db query failed", err)
	}

	var total int64
	if len(rows) > 0 {
		total = rows[0].TotalCount
	}

	now := time.Now()
	items := make([]propertyResponse, 0, len(rows))
	for _, row := range rows {
		resp := propertyResponse{
			ID:        row.ID,
			OwnerID:   row.OwnerID,
			Title:     row.Title,
			Address:   row.Address,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}

		if row.GuestName.Valid {
			guestName := row.GuestName.String
			checkIn := row.CheckIn.Time
			checkOut := row.CheckOut.Time
			resp.GuestName = &guestName
			resp.CurrentCheckIn = &checkIn
			resp.CurrentCheckOut = &checkOut

			if !checkIn.After(now) && checkOut.After(now) {
				resp.Status = "occupied"
			} else {
				resp.Status = "booked"
			}
		} else {
			resp.Status = "vacant"
		}

		items = append(items, resp)
	}

	pages := calcPages(int(total), pageSize)
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
		return errorResponse(c, http.StatusInternalServerError, ErrorCodeInternalServerError, "could not convert userid to string")
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
