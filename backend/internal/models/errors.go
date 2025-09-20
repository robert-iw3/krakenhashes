package models

import "errors"

// Common errors
var (
	ErrNotFound      = errors.New("record not found")
	ErrInvalidInput  = errors.New("invalid input")
	ErrResourceInUse = errors.New("resource is currently in use")
)
