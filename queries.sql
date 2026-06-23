-- name: CreateUser :one
INSERT INTO users (name, email, password_hash, type)
VALUES ($1, $2, $3, $4)
RETURNING id, name, email, type, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, type, address_line_1, address_line_2, postcode, created_at, updated_at
FROM users
WHERE email = $1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token, expires_at, created_at;

-- name: GetSessionByToken :one
SELECT s.id, s.user_id, s.token, s.expires_at, s.created_at,
       u.id AS u_id, u.name AS u_name, u.email AS u_email, u.type AS u_type
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token = $1 AND s.expires_at > now();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = $1;

-- name: CreateProperty :one
INSERT INTO properties (owner_id, title, address)
VALUES ($1, $2, $3)
RETURNING id, owner_id, title, address, created_at, updated_at;

-- name: GetPropertyByID :one
SELECT id, owner_id, title, address, created_at, updated_at
FROM properties
WHERE id = $1;

-- name: ListPropertiesByOwner :many
SELECT id, owner_id, title, address, created_at, updated_at,
       COUNT(*) OVER() AS total_count
FROM properties
WHERE owner_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListCurrentOccupantsByPropertyIDs :many
SELECT DISTINCT ON (property_id)
  property_id, guest_name, check_in, check_out
FROM reservations
WHERE property_id = ANY(@property_ids::uuid[])
  AND check_out > now()
ORDER BY property_id, check_in ASC;

-- name: CreateReservation :one
INSERT INTO reservations (property_id, booked_by, guest_name, check_in, check_out)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, property_id, booked_by, guest_name, check_in, check_out, created_at, updated_at;

-- name: CheckOverlappingReservations :one
SELECT COUNT(*)
FROM reservations
WHERE property_id = $1
  AND (sqlc.narg('exclude_reservation_id')::bigint IS NULL OR id <> sqlc.narg('exclude_reservation_id')::bigint)
  AND check_out > @new_check_in
  AND check_in < @new_check_out;

-- name: GetManagerReservationByID :one
SELECT r.id, r.property_id, p.title AS property_name, r.booked_by, r.guest_name, r.check_in, r.check_out, r.created_at, r.updated_at
FROM reservations r
JOIN properties p ON r.property_id = p.id
WHERE r.id = $1 AND p.owner_id = $2;

-- name: UpdateReservation :one
UPDATE reservations
SET property_id = $2,
    guest_name = $3,
    check_in = $4,
    check_out = $5,
    updated_at = now()
WHERE id = $1
RETURNING id, property_id, booked_by, guest_name, check_in, check_out, created_at, updated_at;

-- name: ListManagerReservationsFiltered :many
SELECT r.id, r.property_id, p.title AS property_name, r.booked_by, r.guest_name, r.check_in, r.check_out, r.created_at, r.updated_at,
       COUNT(*) OVER() AS total_count
FROM reservations r
JOIN properties p ON r.property_id = p.id
WHERE p.owner_id = @owner_id
  AND (sqlc.narg('property_name_filter')::text IS NULL OR p.title ILIKE '%' || sqlc.narg('property_name_filter')::text || '%')
  AND (sqlc.narg('guest_name_filter')::text IS NULL OR r.guest_name ILIKE '%' || sqlc.narg('guest_name_filter')::text || '%')
  AND (sqlc.narg('check_in_from')::timestamptz IS NULL OR r.check_in >= sqlc.narg('check_in_from')::timestamptz)
  AND (sqlc.narg('check_out_to')::timestamptz IS NULL OR r.check_out <= sqlc.narg('check_out_to')::timestamptz)
ORDER BY r.created_at DESC
LIMIT @lim OFFSET @off;
