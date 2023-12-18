package main

import (
	"fmt"
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"net/http"
)

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy",
			"default-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' rsms.me; font-src rsms.me")

		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		next.ServeHTTP(w, r)
	})
}
func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     = r.RemoteAddr
			proto  = r.Proto
			method = r.Method
			uri    = r.URL.RequestURI()
		)

		app.logger.Debug("received request", "ip", ip, "proto", proto, "method", method, "uri", uri)

		next.ServeHTTP(w, r)
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := app.sessionManager.GetBytes(r.Context(), "userID")

		// User has not yet authenticated
		if userID == nil {
			next.ServeHTTP(w, r)
			return
		}

		// If user exists, set context values

		exists, err := app.users.Exists(userID)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		if exists {
			r = contexthelpers.AuthenticateContext(r, userID)
		}

		next.ServeHTTP(w, r)
	})
}

// serverSentMiddleware makes our session library scs work with Server Sent Events (SSE).
// Use this instead of app.sessionManager.LoadAndSave.
// See https://github.com/alexedwards/scs/issues/141#issuecomment-1807075358
func (app *application) serverSentEventMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		cookie, err := r.Cookie(app.sessionManager.Cookie.Name)
		if err == nil {
			token = cookie.Value
		}
		ctx, err := app.sessionManager.Load(r.Context(), token)
		if err != nil {
			app.serverError(w, r, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) commonContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = contexthelpers.SetCurrentPath(r, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
