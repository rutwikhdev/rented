#!/usr/bin/env bash
set -euo pipefail

echo "Starting test database containers..."
docker compose -f docker-compose.test.yml up -d --wait

echo "Running integration tests..."
go test -tags=integration -count=1 -v ./internal/rules/ -run TestNoOverlapRule_Integration 2>&1

echo "Cleaning up..."
docker compose -f docker-compose.test.yml down -v
