package main

import (
	"context"
	"github.com/a-h/templ"
	"github.com/donseba/go-htmx"
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"github.com/myrjola/sheerluck/ui/components"
	"html/template"
	"io"
)

func resolveRoutes(ctx context.Context) []route {
	currentPath := contexthelpers.CurrentPath(ctx)
	routes := []route{
		{
			Href:  "/question-people",
			Title: "Question people",
			Icon:  "/images/talk.svg",
		},
		{
			Href:  "/investigate-scenes",
			Title: "Investigate scenes",
			Icon:  "/images/chalk-outline-murder.svg",
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

func (app *application) Home(_ context.Context, _ *htmx.HxRequestHeader) templ.Component {
	return components.Home()
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

func (app *application) QuestionPeople(ctx context.Context, _ *htmx.HxRequestHeader) templ.Component {
	userID := contexthelpers.AuthenticatedUserID(ctx)
	investigation, _ := app.investigations.Get(ctx, "le-bon", userID)

	chatResponses := make([]components.ChatResponse, len(investigation.Completions))

	for i, completion := range investigation.Completions {
		chatResponses[i] = components.ChatResponse{
			Question: completion.Question,
			Answer:   completion.Answer,
		}
	}

	return components.QuestionPeople(chatResponses)
}

func (app *application) InvestigateScenes(_ context.Context, _ *htmx.HxRequestHeader) templ.Component {
	return components.InvestigateScenes()
}
