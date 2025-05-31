#!/bin/bash
# Script to run all authentication-related tests

set -e

echo "=== Running Authentication Tests ==="
echo

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Check if test database URL is set
if [ -z "$TEST_DATABASE_URL" ]; then
    export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/krakenhashes_test?sslmode=disable"
    echo "Using default TEST_DATABASE_URL: $TEST_DATABASE_URL"
fi

# Set JWT secret for tests
export JWT_SECRET="test-jwt-secret-for-testing-only"

echo "Running JWT tests..."
if go test ./pkg/jwt -v -count=1; then
    echo -e "${GREEN}✓ JWT tests passed${NC}"
else
    echo -e "${RED}✗ JWT tests failed${NC}"
    exit 1
fi

echo
echo "Running Password Validation tests..."
if go test ./pkg/password -v -count=1; then
    echo -e "${GREEN}✓ Password validation tests passed${NC}"
else
    echo -e "${RED}✗ Password validation tests failed${NC}"
    exit 1
fi

echo
echo "Running Authentication Handler tests..."
if go test ./internal/handlers/auth -v -count=1; then
    echo -e "${GREEN}✓ Authentication handler tests passed${NC}"
else
    echo -e "${RED}✗ Authentication handler tests failed${NC}"
    exit 1
fi

echo
echo "Running Integration tests..."
if go test ./internal/integration_test -v -count=1; then
    echo -e "${GREEN}✓ Integration tests passed${NC}"
else
    echo -e "${RED}✗ Integration tests failed${NC}"
    exit 1
fi

echo
echo -e "${GREEN}=== All Authentication Tests Passed ===${NC}"
echo
echo "Test Summary:"
echo "- JWT token validation and generation"
echo "- Password complexity validation"
echo "- Login/logout flows"
echo "- MFA setup and verification (TOTP, email, backup codes)"
echo "- Token management and security"
echo "- Complete user authentication journeys"
echo "- Edge cases and error handling"
echo "- Performance and concurrency scenarios"