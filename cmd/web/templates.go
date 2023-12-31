package main

import (
	"bytes"
	"context"
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"html/template"
	"io"

	"github.com/donseba/go-htmx"
)

func (app *application) executeTemplate(w io.Writer, name string, data any) error {
	t, err := app.parseTemplates()
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, name, data)
}

type baseData struct {
	Nav             template.HTML
	Main            template.HTML
	IsAuthenticated bool
}

func (app *application) Base(ctx context.Context, w io.Writer, h *htmx.HxRequestHeader) error {
	nav := bytes.Buffer{}
	if err := app.Nav(ctx, &nav, h); err != nil {
		return err
	}
	main := bytes.Buffer{}
	if err := app.Home(ctx, &main, h); err != nil {
		return err
	}
	data := baseData{
		Nav:             template.HTML(nav.String()),  //nolint:gosec
		Main:            template.HTML(main.String()), //nolint:gosec
		IsAuthenticated: contexthelpers.IsAuthenticated(ctx),
	}
	if h.HxRequest {
		return app.executeTemplate(w, "body", data)
	}

	return app.executeTemplate(w, "base", data)
}

type homeData struct {
	CSRFToken       string
	IsAuthenticated bool
}

func (app *application) Home(ctx context.Context, w io.Writer, _ *htmx.HxRequestHeader) error {
	data := homeData{
		IsAuthenticated: contexthelpers.IsAuthenticated(ctx),
		CSRFToken:       contexthelpers.CSRFToken(ctx),
	}
	return app.executeTemplate(w, "home", data)
}

type navData struct {
	IsAuthenticated bool
	Routes          []route
}

func (app *application) Nav(ctx context.Context, w io.Writer, _ *htmx.HxRequestHeader) error {
	routes := app.resolveRoutes(contexthelpers.CurrentPath(ctx))
	data := navData{
		IsAuthenticated: contexthelpers.IsAuthenticated(ctx),
		Routes:          routes,
	}
	return app.executeTemplate(w, "nav", data)
}
