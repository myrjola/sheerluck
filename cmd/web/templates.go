package main

import (
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
	Ctx  context.Context
	Nav  template.HTML
	Slot template.HTML
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

type questionPeopleData struct {
	Ctx           context.Context
	ChatResponses []chatResponse
}

func (app *application) QuestionPeople(ctx context.Context, w io.Writer, _ *htmx.HxRequestHeader) error {
	data := questionPeopleData{
		Ctx:           ctx,
		ChatResponses: chatResponses,
	}
	return app.executeTemplate(w, "question-people", data)
}

func (app *application) InvestigateScenes(_ context.Context, w io.Writer, _ *htmx.HxRequestHeader) error {
	return app.executeTemplate(w, "investigate-scenes", nil)
}

func (app *application) ChatResponse(w io.Writer, chatResponse chatResponse) error {
	return app.executeTemplate(w, "chat-response", chatResponse)
}
