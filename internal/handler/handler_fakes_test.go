package handler

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"rented/internal/db"
	"rented/internal/rules"
)

type fakeQueries struct {
	createPropertyFunc                  func(context.Context, db.CreatePropertyParams) (db.Property, error)
	createReservationFunc               func(context.Context, db.CreateReservationParams) (db.Reservation, error)
	createSessionFunc                   func(context.Context, db.CreateSessionParams) (db.Session, error)
	createUserFunc                      func(context.Context, db.CreateUserParams) (db.CreateUserRow, error)
	deleteSessionFunc                   func(context.Context, string) error
	getPropertyByIDFunc                 func(context.Context, pgtype.UUID) (db.Property, error)
	getUserByEmailFunc                  func(context.Context, string) (db.User, error)
	listManagerReservationsFilteredFunc func(
		context.Context,
		db.ListManagerReservationsFilteredParams,
	) ([]db.ListManagerReservationsFilteredRow, error)
	listPropertiesByOwnerFunc func(
		context.Context,
		db.ListPropertiesByOwnerParams,
	) ([]db.ListPropertiesByOwnerRow, error)

	createPropertyParams          db.CreatePropertyParams
	createReservationParams       db.CreateReservationParams
	createSessionParams           db.CreateSessionParams
	createUserParams              db.CreateUserParams
	deleteSessionToken            string
	getPropertyByIDParam          pgtype.UUID
	getUserByEmailParam           string
	listReservationsFilteredParam db.ListManagerReservationsFilteredParams
	listPropertiesByOwnerParam    db.ListPropertiesByOwnerParams
}

func (q *fakeQueries) CreateProperty(ctx context.Context, arg db.CreatePropertyParams) (db.Property, error) {
	q.createPropertyParams = arg
	if q.createPropertyFunc != nil {
		return q.createPropertyFunc(ctx, arg)
	}
	return db.Property{}, nil
}

func (q *fakeQueries) CreateReservation(ctx context.Context, arg db.CreateReservationParams) (db.Reservation, error) {
	q.createReservationParams = arg
	if q.createReservationFunc != nil {
		return q.createReservationFunc(ctx, arg)
	}
	return db.Reservation{}, nil
}

func (q *fakeQueries) CreateSession(ctx context.Context, arg db.CreateSessionParams) (db.Session, error) {
	q.createSessionParams = arg
	if q.createSessionFunc != nil {
		return q.createSessionFunc(ctx, arg)
	}
	return db.Session{}, nil
}

func (q *fakeQueries) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
	q.createUserParams = arg
	if q.createUserFunc != nil {
		return q.createUserFunc(ctx, arg)
	}
	return db.CreateUserRow{}, nil
}

func (q *fakeQueries) DeleteSession(ctx context.Context, token string) error {
	q.deleteSessionToken = token
	if q.deleteSessionFunc != nil {
		return q.deleteSessionFunc(ctx, token)
	}
	return nil
}

func (q *fakeQueries) GetPropertyByID(ctx context.Context, id pgtype.UUID) (db.Property, error) {
	q.getPropertyByIDParam = id
	if q.getPropertyByIDFunc != nil {
		return q.getPropertyByIDFunc(ctx, id)
	}
	return db.Property{}, nil
}

func (q *fakeQueries) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	q.getUserByEmailParam = email
	if q.getUserByEmailFunc != nil {
		return q.getUserByEmailFunc(ctx, email)
	}
	return db.User{}, nil
}

func (q *fakeQueries) ListManagerReservationsFiltered(
	ctx context.Context,
	arg db.ListManagerReservationsFilteredParams,
) ([]db.ListManagerReservationsFilteredRow, error) {
	q.listReservationsFilteredParam = arg
	if q.listManagerReservationsFilteredFunc != nil {
		return q.listManagerReservationsFilteredFunc(ctx, arg)
	}
	return nil, nil
}

func (q *fakeQueries) ListPropertiesByOwner(
	ctx context.Context,
	arg db.ListPropertiesByOwnerParams,
) ([]db.ListPropertiesByOwnerRow, error) {
	q.listPropertiesByOwnerParam = arg
	if q.listPropertiesByOwnerFunc != nil {
		return q.listPropertiesByOwnerFunc(ctx, arg)
	}
	return nil, nil
}

type fakeSessionCache struct {
	setKey string
	setTTL time.Duration
	delKey string
}

func (c *fakeSessionCache) set(_ context.Context, key string, _ any, ttl time.Duration) error {
	c.setKey = key
	c.setTTL = ttl
	return nil
}

func (c *fakeSessionCache) del(_ context.Context, key string) error {
	c.delKey = key
	return nil
}

type fakeRules struct {
	input rules.RuleInput
	errs  []error
}

func (r *fakeRules) Run(_ context.Context, input rules.RuleInput) []error {
	r.input = input
	return r.errs
}
