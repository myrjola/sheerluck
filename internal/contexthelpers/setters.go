package contexthelpers

import (
	"context"
	"net/http"
)

func AuthenticateContext(r *http.Request, userID []byte) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, isAuthenticatedContextKey, true)
	ctx = context.WithValue(ctx, authenticatedUserIDContextKey, userID)
	return r.WithContext(ctx)
}

func SetCurrentPath(r *http.Request, currentPath string) *http.Request {
	ctx := r.Context()
	ctx = context.WithValue(ctx, currentPathContextKey, currentPath)
	return r.WithContext(ctx)
}
