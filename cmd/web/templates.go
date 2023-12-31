package main

import (
	"bytes"
	"context"
	"github.com/donseba/go-htmx"
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"html/template"
	"io"
)

func resolveRoutes(ctx context.Context) []route {
	currentPath := contexthelpers.CurrentPath(ctx)
	routes := []route{
		{
			Href:  "/question-people",
			Title: "Question people",
			Icon:  "talk.svg",
		},
		{
			Href:  "/investigate-scenes",
			Title: "Investigate scenes",
			Icon:  "chalk-outline-murder.svg",
		},
	}

	for i := range routes {
		routes[i].Current = currentPath == routes[i].Href
	}
	return routes
}

var templateFuncMap = template.FuncMap{
	"csrfToken":       contexthelpers.CSRFToken,
	"isAuthenticated": contexthelpers.IsAuthenticated,
	"currentPath":     contexthelpers.CurrentPath,
	"routes":          resolveRoutes,
}

func (app *application) executeTemplate(w io.Writer, name string, data any) error {
	t, err := app.parseTemplates()
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, name, data)
}

type baseData struct {
	Nav  template.HTML
	Main template.HTML
	Ctx  context.Context
}

// TODO: wrap in middleware so that main is passed in as raw HTML, probably need to refactor body to separate template
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
		Nav:  template.HTML(nav.String()),  //nolint:gosec
		Main: template.HTML(main.String()), //nolint:gosec
		Ctx:  ctx,
	}
	if h.HxRequest {
		return app.executeTemplate(w, "body", data)
	}

	return app.executeTemplate(w, "base", data)
}

type homeData struct {
	Ctx context.Context
}

func (app *application) Home(ctx context.Context, w io.Writer, _ *htmx.HxRequestHeader) error {
	data := homeData{
		ctx,
	}
	return app.executeTemplate(w, "home", data)
}

type navData struct {
	Ctx context.Context
}

func (app *application) Nav(ctx context.Context, w io.Writer, _ *htmx.HxRequestHeader) error {
	data := navData{
		Ctx: ctx,
	}
	return app.executeTemplate(w, "nav", data)
}
