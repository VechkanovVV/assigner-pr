#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "Starting integration test environment..."

docker-compose -f "$SCRIPT_DIR/docker-compose.test.yml" down --remove-orphans >/dev/null 2>&1 || true

docker-compose -f "$SCRIPT_DIR/docker-compose.test.yml" up -d --build

echo "Waiting for services to be ready..."
sleep 10

echo "Running integration tests..."
pushd "$PROJECT_ROOT" >/dev/null
export INTEGRATION_DB_HOST=localhost
export INTEGRATION_DB_PORT=5433
export INTEGRATION_DB_USER=test_user
export INTEGRATION_DB_PASSWORD=test_password
export INTEGRATION_DB_NAME=test_db
export INTEGRATION_DB_SSLMODE=disable
TEST_RESULT=0
set +e
go test -v -tags=integration ./internal/integration/... -timeout=5m
TEST_RESULT=$?
set -e
popd >/dev/null
echo "Cleaning up test environment..."
docker-compose -f "$SCRIPT_DIR/docker-compose.test.yml" down
exit $TEST_RESULT