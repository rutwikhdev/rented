package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"rented/internal/db"
)

const pageSize = 50

func (h *Handler) ListProperties(c echo.Context) error {
	session, err := requireSession(c)
	if err != nil {
		return err
	}

	var req struct {
		Page int `json:"page"`
	}
	if err := c.Bind(&req); err != nil {
		h.logError("list properties: failed to bind request", err)
		return invalidRequestBody(c)
	}

	page, offset := normalizePage(req.Page)
	ownerID, err := h.sessionUserID(c, session, "list properties")
	if err != nil {
		return err
	}

	properties, err := h.queries.ListPropertiesByOwner(c.Request().Context(), db.ListPropertiesByOwnerParams{
		OwnerID: ownerID,
		Limit:   int32(pageSize),
		Offset:  offset,
	})
	if err != nil {
		return h.internalError(c, "list properties: db query failed", err)
	}

	var total int64
	if len(properties) > 0 {
		total = properties[0].TotalCount
	}

	pages := calcPages(int(total), pageSize)

	return c.JSON(http.StatusOK, map[string]any{
		"properties": properties,
		"total":      total,
		"page":       page,
		"pages":      pages,
	})
}

type createPropertyRequest struct {
	Title   string `json:"title"`
	Address string `json:"address"`
}

func (h *Handler) CreateProperty(c echo.Context) error {
	session, err := requireSession(c)
	if err != nil {
		return err
	}

	if session.UserType != "manager" {
		return errorResponse(c, http.StatusForbidden, "only managers can create properties")
	}

	var req createPropertyRequest
	if err := c.Bind(&req); err != nil {
		h.logError("create property: failed to bind request", err)
		return invalidRequestBody(c)
	}

	if req.Title == "" || req.Address == "" {
		return errorResponse(c, http.StatusBadRequest, "title and address are required")
	}

	ownerID, err := h.sessionUserID(c, session, "create property")
	if err != nil {
		return err
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
