package main

import (
	"fmt"
	"github.com/justinas/nosurf"
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"github.com/myrjola/sheerluck/internal/random"
	"net/http"
)

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate a random nonce for use in CSP and set it in the context so that it can be added to the script tags.
		cspNonce, err := random.RandomLetters(24)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		csp := fmt.Sprintf(`script-src 'nonce-%s' 'strict-dynamic' https: http:;
style-src 'nonce-%s' 'strict-dynamic' https: http:;
object-src 'none';
base-uri 'none';`, cspNonce, cspNonce)

		w.Header().Set("Content-Security-Policy", csp)
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		r = contexthelpers.SetCSPNonce(r, cspNonce)

		next.ServeHTTP(w, r)
	})
}

func cacheForeverHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

		next.ServeHTTP(w, r)
	})
}

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			proto  = r.Proto
			method = r.Method
			uri    = r.URL.RequestURI()
		)

		app.logger.Debug("received request", "proto", proto, "method", method, "uri", uri)

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
		userID := app.sessionManager.GetBytes(r.Context(), string(userIDSessionKey))

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

// mustAuthenticate redirects the user to the home page if they are not authenticated.
func (app *application) mustAuthenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAuthenticated := contexthelpers.IsAuthenticated(r.Context())
		if !isAuthenticated {
			http.Redirect(w, r, "/", http.StatusSeeOther)
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

func commonContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = contexthelpers.SetCurrentPath(r, r.URL.Path)
		r = contexthelpers.SetCSRFToken(r, nosurf.Token(r))
		next.ServeHTTP(w, r)
	})
}

// noSurf implements CSRF protection using https://github.com/justinas/nosurf
func noSurf(next http.Handler) http.Handler {
	csrfHandler := nosurf.New(next)
	csrfHandler.SetBaseCookie(http.Cookie{
		HttpOnly: true,
		Path:     "/",
		Secure:   true,
	})
	// TODO: Enable CSRF protection for the JSON API endpoints.
	csrfHandler.ExemptPaths("/api/registration/start", "/api/registration/finish", "/api/login/start", "/api/login/finish")

	return csrfHandler
}
