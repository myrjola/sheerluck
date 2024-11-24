package main

import (
	"github.com/myrjola/sheerluck/internal/errors"
	"log/slog"
	"net/http"
)

func (app *application) serverError(w http.ResponseWriter, r *http.Request, err error) {
	var (
		method = r.Method
		uri    = r.URL.RequestURI()
	)

	app.logger.LogAttrs(r.Context(), slog.LevelError, "server error",
		slog.String("method", method), slog.String("uri", uri), errors.SlogError(err))
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
