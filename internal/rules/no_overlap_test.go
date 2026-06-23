//go:build integration

package rules

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"rented/internal/db"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5433/rented-test?sslmode=disable"
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration tests: failed to connect to database: %v\n", err)
		fmt.Fprintf(os.Stderr, "run 'docker compose -f docker-compose.test.yml up -d --wait' first\n")
		os.Exit(0)
	}

	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "integration tests: database not reachable: %v\n", err)
		os.Exit(0)
	}

	schema, err := os.ReadFile("../../schema.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration tests: failed to read schema file: %v\n", err)
		os.Exit(0)
	}

	if _, err := pool.Exec(ctx, string(schema)); err != nil {
		fmt.Fprintf(os.Stderr, "integration tests: failed to run schema: %v\n", err)
		os.Exit(0)
	}

	testPool = pool
	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func TestNoOverlapRule_Integration(t *testing.T) {
	if testPool == nil {
		t.Skip("integration tests require a database connection")
	}

	existingCheckIn := time.Date(2028, 7, 1, 10, 0, 0, 0, time.UTC)
	existingCheckOut := time.Date(2028, 7, 5, 9, 30, 0, 0, time.UTC)

	tests := []struct {
		name            string
		checkIn         time.Time
		checkOut        time.Time
		expectErr       bool
		excludeExisting bool
	}{
		{
			name:      "no overlap - entirely before",
			checkIn:   time.Date(2028, 6, 28, 10, 0, 0, 0, time.UTC),
			checkOut:  time.Date(2028, 7, 1, 9, 30, 0, 0, time.UTC),
			expectErr: false,
		},
		{
			name:      "no overlap - entirely after",
			checkIn:   time.Date(2028, 7, 5, 10, 0, 0, 0, time.UTC),
			checkOut:  time.Date(2028, 7, 8, 9, 30, 0, 0, time.UTC),
			expectErr: false,
		},
		{
			name:      "adjacent - ends exactly when existing starts",
			checkIn:   time.Date(2028, 6, 28, 10, 0, 0, 0, time.UTC),
			checkOut:  time.Date(2028, 7, 1, 10, 0, 0, 0, time.UTC),
			expectErr: false,
		},
		{
			name:      "adjacent - starts exactly when existing ends",
			checkIn:   time.Date(2028, 7, 5, 9, 30, 0, 0, time.UTC),
			checkOut:  time.Date(2028, 7, 8, 9, 30, 0, 0, time.UTC),
			expectErr: false,
		},
		{
			name:      "overlaps start of existing",
			checkIn:   time.Date(2028, 6, 30, 10, 0, 0, 0, time.UTC),
			checkOut:  time.Date(2028, 7, 3, 9, 30, 0, 0, time.UTC),
			expectErr: true,
		},
		{
			name:      "overlaps end of existing",
			checkIn:   time.Date(2028, 7, 3, 10, 0, 0, 0, time.UTC),
			checkOut:  time.Date(2028, 7, 8, 9, 30, 0, 0, time.UTC),
			expectErr: true,
		},
		{
			name:      "contained within existing",
			checkIn:   time.Date(2028, 7, 2, 10, 0, 0, 0, time.UTC),
			checkOut:  time.Date(2028, 7, 4, 9, 30, 0, 0, time.UTC),
			expectErr: true,
		},
		{
			name:      "contains existing entirely",
			checkIn:   time.Date(2028, 6, 30, 10, 0, 0, 0, time.UTC),
			checkOut:  time.Date(2028, 7, 8, 9, 30, 0, 0, time.UTC),
			expectErr: true,
		},
		{
			name:      "same dates as existing returns error",
			checkIn:   existingCheckIn,
			checkOut:  existingCheckOut,
			expectErr: true,
		},
		{
			name:            "exclude existing reservation allows same dates",
			checkIn:         existingCheckIn,
			checkOut:        existingCheckOut,
			expectErr:       false,
			excludeExisting: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tx, err := testPool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)

			q := db.New(tx)

			user, err := q.CreateUser(ctx, db.CreateUserParams{
				Name:         "Test Manager",
				Email:        fmt.Sprintf("manager-%d@test.com", time.Now().UnixNano()),
				PasswordHash: "hash",
				Type:         "manager",
			})
			if err != nil {
				t.Fatal(err)
			}

			property, err := q.CreateProperty(ctx, db.CreatePropertyParams{
				OwnerID: user.ID,
				Title:   "Test Property",
				Address: "123 Test St",
			})
			if err != nil {
				t.Fatal(err)
			}

			existing, err := q.CreateReservation(ctx, db.CreateReservationParams{
				PropertyID: property.ID,
				BookedBy:   user.ID,
				GuestName:  "Existing Guest",
				CheckIn:    pgtype.Timestamptz{Time: existingCheckIn, Valid: true},
				CheckOut:   pgtype.Timestamptz{Time: existingCheckOut, Valid: true},
			})
			if err != nil {
				t.Fatal(err)
			}

			rule := NewNoOverlapRule(q)
			input := RuleInput{
				PropertyID: property.ID,
				CheckIn:    tt.checkIn,
				CheckOut:   tt.checkOut,
			}
			if tt.excludeExisting {
				input.ExcludeReservationID = existing.ID
			}

			err = rule.Check(ctx, input)
			if tt.expectErr {
				if err == nil {
					t.Fatal("expected ErrOverlappingReservation but got nil")
				}
				if !errors.Is(err, ErrOverlappingReservation) {
					t.Fatalf("expected ErrOverlappingReservation, got %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
