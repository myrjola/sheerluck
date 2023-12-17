package main

type contextKey string

const isAuthenticatedContextKey = contextKey("isAuthenticated")
const authenticatedUserIDContextKey = contextKey("authenticatedUserID")
