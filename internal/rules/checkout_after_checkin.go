package rules

import (
	"context"
	"errors"
)

var ErrCheckoutBeforeCheckin = errors.New("check-out must be after check-in")

type CheckoutAfterCheckinRule struct{}

func NewCheckoutAfterCheckinRule() *CheckoutAfterCheckinRule {
	return &CheckoutAfterCheckinRule{}
}

func (r *CheckoutAfterCheckinRule) Name() string {
	return "checkout_after_checkin"
}

func (r *CheckoutAfterCheckinRule) Check(ctx context.Context, input RuleInput) error {
	if !input.CheckOut.After(input.CheckIn) {
		return ErrCheckoutBeforeCheckin
	}
	return nil
}
