package main

import (
	"context"
	"github.com/a-h/templ"
	"github.com/donseba/go-htmx"
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"github.com/myrjola/sheerluck/ui/components"
)

func (app *application) Home(_ context.Context, _ *htmx.HxRequestHeader) templ.Component {
	return components.Home()
}

type navData struct {
	Ctx context.Context
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

func (app *application) CaseHome(_ context.Context, _ *htmx.HxRequestHeader) templ.Component {
	// TODO: How do I get the case here? Should I consider another signature for these components including the request and response?

	return components.CaseHome()
}
