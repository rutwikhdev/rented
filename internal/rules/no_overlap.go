package rules

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"

	"rented/internal/db"
)

var ErrOverlappingReservation = errors.New("this property already has a reservation that overlaps with the requested dates")

type NoOverlapRule struct {
	queries *db.Queries
}

func NewNoOverlapRule(queries *db.Queries) *NoOverlapRule {
	return &NoOverlapRule{queries: queries}
}

func (r *NoOverlapRule) Name() string {
	return "no_overlap"
}

func (r *NoOverlapRule) Check(ctx context.Context, input RuleInput) error {
	var checkIn, checkOut pgtype.Timestamptz
	checkIn.Time = input.CheckIn
	checkIn.Valid = true
	checkOut.Time = input.CheckOut
	checkOut.Valid = true

	count, err := r.queries.CheckOverlappingReservations(ctx, db.CheckOverlappingReservationsParams{
		PropertyID:  input.PropertyID,
		NewCheckIn:  checkIn,
		NewCheckOut: checkOut,
	})
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrOverlappingReservation
	}
	return nil
}
