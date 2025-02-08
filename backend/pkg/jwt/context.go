package jwt

import (
	"context"
)

type contextKey string

const userIDKey contextKey = "user_id"

// WithUserID adds a user ID to the context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// GetUserID retrieves the user ID from the context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}
