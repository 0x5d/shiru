package api

import (
	"context"

	"github.com/google/uuid"
)

type contextKey int

const userIDKey contextKey = iota

func contextWithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

func userIDFromContext(ctx context.Context) uuid.UUID {
	id, _ := ctx.Value(userIDKey).(uuid.UUID)
	return id
}
