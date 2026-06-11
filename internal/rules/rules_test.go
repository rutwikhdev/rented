package rules

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCheckoutAfterCheckinRule_Check(t *testing.T) {
	now := time.Now()
	rule := NewCheckoutAfterCheckinRule()

	tests := []struct {
		name    string
		input   RuleInput
		wantErr error
	}{
		{
			name: "checkout after checkin",
			input: RuleInput{
				CheckIn:  now,
				CheckOut: now.Add(time.Hour),
			},
		},
		{
			name: "checkout equals checkin",
			input: RuleInput{
				CheckIn:  now,
				CheckOut: now,
			},
			wantErr: ErrCheckoutBeforeCheckin,
		},
		{
			name: "checkout before checkin",
			input: RuleInput{
				CheckIn:  now,
				CheckOut: now.Add(-time.Hour),
			},
			wantErr: ErrCheckoutBeforeCheckin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Check(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Check() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckInNotInPastRule_Check(t *testing.T) {
	rule := NewCheckInNotInPastRule()

	tests := []struct {
		name    string
		input   RuleInput
		wantErr error
	}{
		{
			name: "future checkin",
			input: RuleInput{
				CheckIn: time.Now().Add(time.Hour),
			},
		},
		{
			name: "past checkin",
			input: RuleInput{
				CheckIn: time.Now().Add(-time.Hour),
			},
			wantErr: ErrCheckInInPast,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := rule.Check(context.Background(), tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Check() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_Run(t *testing.T) {
	firstErr := errors.New("first error")
	secondErr := errors.New("second error")

	tests := []struct {
		name     string
		rules    []Rule
		wantErrs []error
	}{
		{
			name: "all rules pass",
			rules: []Rule{
				testRule{name: "first"},
				testRule{name: "second"},
			},
		},
		{
			name: "collects errors in order",
			rules: []Rule{
				testRule{name: "first", err: firstErr},
				testRule{name: "pass"},
				testRule{name: "second", err: secondErr},
			},
			wantErrs: []error{firstErr, secondErr},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewEngine(tt.rules...)
			got := engine.Run(context.Background(), RuleInput{})

			if len(got) != len(tt.wantErrs) {
				t.Fatalf("Run() returned %d errors, want %d", len(got), len(tt.wantErrs))
			}

			for i, want := range tt.wantErrs {
				if !errors.Is(got[i], want) {
					t.Fatalf("Run() error[%d] = %v, want %v", i, got[i], want)
				}
			}
		})
	}
}

type testRule struct {
	name string
	err  error
}

func (r testRule) Name() string {
	return r.name
}

func (r testRule) Check(ctx context.Context, input RuleInput) error {
	return r.err
}
