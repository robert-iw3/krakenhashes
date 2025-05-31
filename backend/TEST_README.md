# KrakenHashes Backend Test Suite

## Overview

This test suite provides comprehensive coverage of the KrakenHashes backend authentication system, including unit tests, integration tests, and security validation.

## Test Coverage

### 🔐 Authentication Tests (`internal/handlers/auth/`)

**Core Authentication**
- ✅ Login/logout flows
- ✅ JWT token generation and validation
- ✅ Session management
- ✅ Cookie handling and security attributes
- ✅ Multi-device token support

**Multi-Factor Authentication (MFA)**
- ✅ TOTP authenticator setup and verification
- ✅ Email MFA code generation and validation
- ✅ Backup code generation and consumption
- ✅ MFA session management
- ✅ Max attempts and rate limiting

**Token Security**
- ✅ Token isolation between users
- ✅ Token replay prevention
- ✅ Invalid token handling
- ✅ Concurrent token operations

### 🔑 JWT Package Tests (`pkg/jwt/`)

- ✅ Token generation with different user roles
- ✅ Token validation (valid, expired, malformed)
- ✅ Role extraction from tokens
- ✅ Secure token generation
- ✅ Admin role validation
- ✅ Empty JWT secret handling

### 🛡️ Password Validation Tests (`pkg/password/`)

- ✅ Password complexity requirements
- ✅ Unicode character support
- ✅ Validation error handling
- ✅ Complexity description generation
- ✅ Edge cases and boundary conditions
- ✅ Performance benchmarks

### 🔄 Integration Tests (`internal/integration_test/`)

**Complete User Journeys**
- ✅ New user registration → MFA setup → login → logout
- ✅ Admin user with elevated security requirements
- ✅ Mobile app user with email MFA preference

**Security Scenarios**
- ✅ Brute force protection
- ✅ Session hijacking prevention
- ✅ Timing attack resistance
- ✅ Concurrent access patterns

**Edge Cases**
- ✅ Password complexity edge cases
- ✅ Username validation edge cases
- ✅ Session expiration handling
- ✅ MFA timing boundaries (TOTP windows)

**Performance Testing**
- ✅ Rapid login attempts
- ✅ Concurrent token validations
- ✅ Load testing scenarios

## Running Tests

### All Authentication Tests
```bash
# Run complete auth test suite
make test-auth

# Or using the script
./scripts/test-auth.sh
```

### Individual Test Suites
```bash
# JWT tests only
go test ./pkg/jwt -v

# Password validation tests only
go test ./pkg/password -v

# Auth handler tests only
go test ./internal/handlers/auth -v

# Integration tests only
make test-integration
```

### With Coverage
```bash
# Generate coverage report
make test-coverage

# View coverage report
open coverage.html
```

## Test Environment Setup

### Required Environment Variables
```bash
# JWT secret for token signing
export JWT_SECRET="test-jwt-secret-for-testing-only"

# Test database URL (optional)
export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/krakenhashes_test?sslmode=disable"
```

### Test Database
The tests use a separate test database that is automatically cleaned between tests. Each test gets a fresh database state.

## Test Structure

### Test Utilities (`internal/testutil/`)

**Database Setup**
- `SetupTestDB()` - Creates isolated test database
- `CreateTestUser()` - Helper for creating test users
- Database cleanup and transaction management

**Mocks and Fixtures**
- `MockEmailService` - Email service mock for MFA testing
- `MockTLSProvider` - TLS provider mock for certificate testing
- Test user fixtures and constants

**Helper Functions**
- `MakeAuthenticatedRequest()` - Creates requests with valid auth tokens
- `AssertJSONResponse()` - Validates HTTP responses
- `AssertCookieSet()` - Validates cookie attributes

### Test Patterns

**Unit Tests**
- Test individual functions in isolation
- Mock external dependencies
- Focus on business logic and edge cases

**Integration Tests**
- Test complete workflows end-to-end
- Use real database connections
- Validate full request/response cycles

**Security Tests**
- Validate security controls and protections
- Test attack scenarios and edge cases
- Ensure proper error handling

## Security Testing

### Attack Scenarios Covered

1. **Authentication Bypass**
   - Invalid credentials
   - Missing authentication
   - Token tampering

2. **Session Management**
   - Session fixation
   - Session hijacking
   - Token replay attacks

3. **Multi-Factor Authentication**
   - MFA bypass attempts
   - Code reuse prevention
   - Timing attacks on TOTP

4. **Rate Limiting**
   - Brute force attacks
   - Rapid request patterns
   - Resource exhaustion

5. **Input Validation**
   - Malformed requests
   - Boundary conditions
   - Unicode handling

## Performance Benchmarks

The test suite includes performance benchmarks for:

- Password validation operations
- JWT token generation/validation
- Database operations
- Concurrent authentication requests

Run benchmarks with:
```bash
go test -bench=. ./pkg/password
go test -bench=. ./pkg/jwt
```

## Test Metrics

### Coverage Goals
- **Unit Tests**: >90% line coverage
- **Integration Tests**: All critical user paths
- **Security Tests**: All attack vectors

### Test Categories
- **Fast Tests** (<100ms): Unit tests, mocks
- **Medium Tests** (<1s): Database integration
- **Slow Tests** (<10s): Full workflow integration

## Continuous Integration

### Pre-commit Checks
```bash
# Run all auth tests
make test-auth

# Check test coverage
make test-coverage

# Lint and format
go fmt ./...
go vet ./...
```

### CI Pipeline
The test suite is designed to run in CI/CD pipelines with:
- Parallel test execution
- Database isolation
- Comprehensive error reporting
- Coverage tracking

## Troubleshooting

### Common Issues

**Database Connection Errors**
```bash
# Ensure PostgreSQL is running
docker-compose up postgres

# Check connection
psql $TEST_DATABASE_URL -c "SELECT 1"
```

**JWT Secret Issues**
```bash
# Set JWT secret
export JWT_SECRET="test-jwt-secret-for-testing-only"
```

**Port Conflicts**
- Tests use ephemeral ports by default
- No external port dependencies required

### Debug Mode
```bash
# Run with verbose output
go test -v -count=1 ./internal/handlers/auth

# Run specific test
go test -v -run TestLoginHandler ./internal/handlers/auth
```

## Contributing

### Adding New Tests

1. **Unit Tests**: Add to appropriate `*_test.go` file
2. **Integration Tests**: Add to `internal/integration_test/`
3. **Follow naming conventions**: `TestFunctionName` for tests
4. **Use test utilities**: Leverage existing fixtures and helpers
5. **Document test scenarios**: Clear test names and comments

### Test Requirements

- All tests must be deterministic
- Clean up resources after test completion
- Use appropriate assertion libraries
- Include both positive and negative test cases
- Test edge cases and error conditions

## Security Notes

⚠️ **Important**: These tests use fixed secrets and test data. Never use test credentials or secrets in production environments.

The test suite validates security controls but should be supplemented with:
- Professional security audits
- Penetration testing
- Code security analysis
- Dependency vulnerability scanning