package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"

	"rented/internal/db"
	"rented/internal/utils"
)

const sessionTTL = 7 * 24 * time.Hour

type SessionData struct {
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	UserEmail string `json:"user_email"`
	UserType  string `json:"user_type"`
}

type signupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Type     string `json:"type"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authenticatedUser struct {
	ID    int64
	Name  string
	Email string
	Type  string
}

func (h *Handler) Signup(c echo.Context) error {
	var req signupRequest
	if err := c.Bind(&req); err != nil {
		h.logError("signup: failed to bind request", err)
		return invalidRequestBody(c)
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		return errorResponse(c, http.StatusBadRequest, "name, email, and password are required")
	}

	req.Email = utils.SanitizeEmail(req.Email)

	if err := utils.ValidateEmail(req.Email); err != nil {
		return errorResponse(c, http.StatusBadRequest, "invalid email format")
	}

	if err := utils.ValidatePassword(req.Password); err != nil {
		return errorResponse(c, http.StatusBadRequest, "password must be at least 8 characters")
	}

	userType := req.Type
	if userType == "" {
		userType = "manager"
	}
	if userType != "guest" && userType != "manager" {
		return errorResponse(c, http.StatusBadRequest, "type must be guest or manager")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return h.internalError(c, "signup: failed to hash password", err)
	}

	user, err := h.queries.CreateUser(c.Request().Context(), db.CreateUserParams{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
		Type:         userType,
	})
	if err != nil {
		h.logError("signup: failed to create user", err, "email", req.Email)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errorResponse(c, http.StatusConflict, "email already exists")
		}
		return errorResponse(c, http.StatusInternalServerError, "internal server error")
	}

	session, err := h.createAuthenticatedSession(c.Request().Context(), authenticatedUser{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Type:  user.Type,
	})
	if err != nil {
		return h.internalError(c, "signup: failed to create session", err, "user_id", user.ID)
	}

	return c.JSON(http.StatusCreated, session)
}

func (h *Handler) Login(c echo.Context) error {
	var req loginRequest
	if err := c.Bind(&req); err != nil {
		h.logError("login: failed to bind request", err)
		return invalidRequestBody(c)
	}

	if req.Email == "" || req.Password == "" {
		return errorResponse(c, http.StatusBadRequest, "email and password are required")
	}

	req.Email = utils.SanitizeEmail(req.Email)

	user, err := h.queries.GetUserByEmail(c.Request().Context(), req.Email)
	if err != nil {
		h.logError("login: failed to get user", err, "email", req.Email)
		if errors.Is(err, pgx.ErrNoRows) {
			return errorResponse(c, http.StatusUnauthorized, "invalid credentials")
		}
		return errorResponse(c, http.StatusInternalServerError, "internal server error")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return errorResponse(c, http.StatusUnauthorized, "invalid credentials")
	}

	session, err := h.createAuthenticatedSession(c.Request().Context(), authenticatedUser{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
		Type:  user.Type,
	})
	if err != nil {
		return h.internalError(c, "login: failed to create session", err, "user_id", user.ID)
	}

	return c.JSON(http.StatusOK, session)
}

func (h *Handler) Logout(c echo.Context) error {
	token := ExtractToken(c)
	if token == "" {
		return errorResponse(c, http.StatusUnauthorized, "no token provided")
	}

	if err := h.queries.DeleteSession(c.Request().Context(), token); err != nil {
		h.logError("logout: failed to delete session from db", err)
	}
	if h.redis != nil {
		if err := h.redis.del(c.Request().Context(), "session:"+token); err != nil {
			h.logError("logout: failed to delete session from redis", err)
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *Handler) createAuthenticatedSession(ctx context.Context, user authenticatedUser) (SessionData, error) {
	token, err := generateToken()
	if err != nil {
		return SessionData{}, err
	}

	expiresAt := time.Now().Add(sessionTTL)
	if _, err := h.queries.CreateSession(ctx, db.CreateSessionParams{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: pgTimestamp(expiresAt),
	}); err != nil {
		return SessionData{}, err
	}

	session := SessionData{
		Token:     token,
		UserID:    strconv.FormatInt(user.ID, 10),
		UserName:  user.Name,
		UserEmail: user.Email,
		UserType:  user.Type,
	}

	h.cacheSession(ctx, token, session, expiresAt)
	return session, nil
}

func (h *Handler) cacheSession(ctx context.Context, token string, session SessionData, expiresAt time.Time) {
	data, err := json.Marshal(session)
	if err != nil {
		h.logError("cacheSession: failed to marshal session", err)
		return
	}
	ttl := time.Until(expiresAt)
	if h.redis == nil {
		return
	}
	if err := h.redis.set(ctx, "session:"+token, data, ttl); err != nil {
		h.logError("cacheSession: failed to set in redis", err)
	}
}

func ExtractToken(c echo.Context) string {
	auth := c.Request().Header.Get("Authorization")
	if len(auth) > 7 && auth[:7] == "Bearer " {
		return auth[7:]
	}
	return ""
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
