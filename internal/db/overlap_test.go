//go:build integration

package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
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

func TestExclusionConstraint(t *testing.T) {
	if testPool == nil {
		t.Skip("integration tests require a database connection")
	}

	existingCheckIn := time.Date(2028, 7, 1, 10, 0, 0, 0, time.UTC)
	existingCheckOut := time.Date(2028, 7, 5, 9, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		checkIn   time.Time
		checkOut  time.Time
		expectErr bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			tx, err := testPool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)

			q := New(tx)

			user, err := q.CreateUser(ctx, CreateUserParams{
				Name:         "Test Manager",
				Email:        fmt.Sprintf("manager-%d@test.com", time.Now().UnixNano()),
				PasswordHash: "hash",
				Type:         "manager",
			})
			if err != nil {
				t.Fatal(err)
			}

			property, err := q.CreateProperty(ctx, CreatePropertyParams{
				OwnerID: user.ID,
				Title:   "Test Property",
				Address: "123 Test St",
			})
			if err != nil {
				t.Fatal(err)
			}

			_, err = q.CreateReservation(ctx, CreateReservationParams{
				PropertyID: property.ID,
				BookedBy:   user.ID,
				GuestName:  "Existing Guest",
				CheckIn:    pgtype.Timestamptz{Time: existingCheckIn, Valid: true},
				CheckOut:   pgtype.Timestamptz{Time: existingCheckOut, Valid: true},
			})
			if err != nil {
				t.Fatal(err)
			}

			_, err = q.CreateReservation(ctx, CreateReservationParams{
				PropertyID: property.ID,
				BookedBy:   user.ID,
				GuestName:  "New Guest",
				CheckIn:    pgtype.Timestamptz{Time: tt.checkIn, Valid: true},
				CheckOut:   pgtype.Timestamptz{Time: tt.checkOut, Valid: true},
			})

			if tt.expectErr {
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != "23P01" {
					t.Fatalf("expected exclusion violation (23P01), got %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}
