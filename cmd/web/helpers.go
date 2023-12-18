package main

import (
	"github.com/a-h/templ"
	"github.com/myrjola/sheerluck/ui/html/layout"
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

func (app *application) renderPage(c templ.Component, w http.ResponseWriter, r *http.Request) error {
	var (
		component = c
	)

	// If this is not a HTMX boosted request, wrap the component with the base.
	// See https://htmx.org/attributes/hx-boost/
	if r.Header.Get("Hx-Boosted") != "true" {
		component = layout.Base(component)
	}

	return component.Render(r.Context(), w)
}
