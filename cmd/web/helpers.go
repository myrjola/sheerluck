package main

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
		trace  = string(debug.Stack())
	)

	app.logger.Error(err.Error(), "method", method, "uri", uri, "trace", trace)
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func (app *application) clientError(w http.ResponseWriter, r *http.Request, status int) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	app.logger.Debug(http.StatusText(status), "method", method, "uri", uri, slog.Any("formdata", r.Form))
	http.Error(w, http.StatusText(status), status)
}

func (app *application) notFound(w http.ResponseWriter, r *http.Request) {
	app.clientError(w, r, http.StatusNotFound)
}

func (app *application) isAuthenticated(r *http.Request) bool {
	isAuthenticated, ok := r.Context().Value(isAuthenticatedContextKey).(bool)
	if !ok {
		return false
	}

	return isAuthenticated
}

func (app *application) getAuthenticatedUserID(r *http.Request) []byte {
	userID, ok := r.Context().Value(authenticatedUserIDContextKey).([]byte)
	if !ok {
		return nil
	}

	return userID
}
