package repository

import "errors"

var (
	// ErrNotFound is returned when a requested resource is not found
	ErrNotFound = errors.New("resource not found")

	// ErrInvalidStatus is returned when an invalid status is provided
	ErrInvalidStatus = errors.New("invalid status")

	// ErrInvalidVoucher is returned when a voucher is invalid or expired
	ErrInvalidVoucher = errors.New("invalid or expired voucher")

	// ErrVoucherAlreadyUsed is returned when a single-use voucher has already been used
	ErrVoucherAlreadyUsed = errors.New("voucher has already been used")

	// ErrVoucherDeactivated is returned when attempting to use a deactivated voucher
	ErrVoucherDeactivated = errors.New("voucher has been deactivated")

	// ErrVoucherExpired is returned when attempting to use an expired voucher
	ErrVoucherExpired = errors.New("voucher has expired")

	// ErrDuplicateToken is returned when attempting to create an agent with a duplicate token
	ErrDuplicateToken = errors.New("agent token already exists")

	// ErrInvalidToken is returned when an invalid token is provided
	ErrInvalidToken = errors.New("invalid token")

	// ErrAgentNotFound is returned when an agent is not found
	ErrAgentNotFound = errors.New("agent not found")

	// ErrInvalidHardware is returned when invalid hardware information is provided
	ErrInvalidHardware = errors.New("invalid hardware information")

	// ErrInvalidMetrics is returned when invalid metrics are provided
	ErrInvalidMetrics = errors.New("invalid metrics")

	// ErrDuplicateRecord is returned when attempting to create a record that violates a unique constraint
	ErrDuplicateRecord = errors.New("duplicate record")
)
