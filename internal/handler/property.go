package handler

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"

	"rented/internal/db"
)

const pageSize = 50

func (h *Handler) ListProperties(c echo.Context) error {
	session, ok := c.Get("session").(*SessionData)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, "unauthorized")
	}

	var req struct {
		Page int `json:"page"`
	}
	if err := c.Bind(&req); err != nil {
		h.logger.Error("list properties: failed to bind request", err)
		return invalidRequestBody(c)
	}

	if req.Page < 1 {
		req.Page = 1
	}
	offset := (req.Page - 1) * pageSize

	var ownerID pgtype.UUID
	if err := ownerID.Scan(session.UserID); err != nil {
		h.logger.Error("list properties: invalid session user id", err)
		return errorResponse(c, http.StatusUnauthorized, "unauthorized")
	}

	properties, err := h.queries.ListPropertiesByOwner(c.Request().Context(), db.ListPropertiesByOwnerParams{
		OwnerID: ownerID,
		Limit:   int32(pageSize),
		Offset:  int32(offset),
	})
	if err != nil {
		return h.internalError(c, "list properties: db query failed", err)
	}

	total, err := h.queries.CountPropertiesByOwner(c.Request().Context(), ownerID)
	if err != nil {
		return h.internalError(c, "list properties: count failed", err)
	}

	pages := calcPages(int(total), pageSize)

	return c.JSON(http.StatusOK, map[string]any{
		"properties": properties,
		"total":      total,
		"page":       req.Page,
		"pages":      pages,
	})
}

type createPropertyRequest struct {
	Title   string `json:"title"`
	Address string `json:"address"`
}

func (h *Handler) CreateProperty(c echo.Context) error {
	session, ok := c.Get("session").(*SessionData)
	if !ok {
		return errorResponse(c, http.StatusUnauthorized, "unauthorized")
	}

	if session.UserType != "manager" {
		return errorResponse(c, http.StatusForbidden, "only managers can create properties")
	}

	var req createPropertyRequest
	if err := c.Bind(&req); err != nil {
		h.logger.Error("create property: failed to bind request", err)
		return invalidRequestBody(c)
	}

	if req.Title == "" || req.Address == "" {
		return errorResponse(c, http.StatusBadRequest, "title and address are required")
	}

	var ownerID pgtype.UUID
	if err := ownerID.Scan(session.UserID); err != nil {
		h.logger.Error("create property: invalid session user id", err)
		return errorResponse(c, http.StatusUnauthorized, "unauthorized")
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
