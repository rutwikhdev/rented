package rules

import (
	"context"
	"errors"
	"time"
)

var ErrCheckInInPast = errors.New("check-in cannot be in the past")

type CheckInNotInPastRule struct{}

func NewCheckInNotInPastRule() *CheckInNotInPastRule {
	return &CheckInNotInPastRule{}
}

func (r *CheckInNotInPastRule) Name() string {
	return "checkin_not_in_past"
}

func (r *CheckInNotInPastRule) Check(ctx context.Context, input RuleInput) error {
	if input.CheckIn.Before(time.Now()) {
		return ErrCheckInInPast
	}
	return nil
}
