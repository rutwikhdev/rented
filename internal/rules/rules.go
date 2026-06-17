package rules

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type RuleInput struct {
	PropertyID           pgtype.UUID
	CheckIn              time.Time
	CheckOut             time.Time
	ExcludeReservationID int64
}

type Rule interface {
	Name() string
	Check(ctx context.Context, input RuleInput) error
}

type Engine struct {
	rules []Rule
}

func NewEngine(rules ...Rule) *Engine {
	return &Engine{rules: rules}
}

func (e *Engine) Run(ctx context.Context, input RuleInput) []error {
	var errs []error
	for _, r := range e.rules {
		if err := r.Check(ctx, input); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
