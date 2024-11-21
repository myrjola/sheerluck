package main

import (
	"github.com/a-h/templ"
	"github.com/donseba/go-htmx"
	"github.com/myrjola/sheerluck/db"
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"github.com/myrjola/sheerluck/ui/components"
	"net/http"
)

type BaseTemplateData struct {
	Authenticated bool
}

func newBaseTemplateData(r *http.Request) BaseTemplateData {
	return BaseTemplateData{
		Authenticated: contexthelpers.IsAuthenticated(r.Context()),
	}
}

func (app *application) Home(_ http.ResponseWriter, _ *http.Request, _ *htmx.HxRequestHeader) (*templ.Component, error) {
	home := components.Home()
	return &home, nil
}

func (app *application) QuestionPeople(_ http.ResponseWriter, r *http.Request, _ *htmx.HxRequestHeader) (*templ.Component, error) {
	ctx := r.Context()
	userID := contexthelpers.AuthenticatedUserID(ctx)
	investigationTargetID := r.PathValue("investigationTargetID")
	investigation, _ := app.investigations.Get(ctx, investigationTargetID, userID)

	chatResponses := make([]components.ChatResponse, len(investigation.Completions))

	for i, completion := range investigation.Completions {
		chatResponses[i] = components.ChatResponse{
			Question: completion.Question,
			Answer:   completion.Answer,
		}
	}

	questionPeople := components.QuestionPeople(chatResponses)

	return &questionPeople, nil
}

func (app *application) InvestigateScenes(_ http.ResponseWriter, r *http.Request, _ *htmx.HxRequestHeader) (*templ.Component, error) {
	investigateScenes := components.InvestigateScenes()
	return &investigateScenes, nil
}

func (app *application) CaseView(w http.ResponseWriter, r *http.Request, _ *htmx.HxRequestHeader) (*templ.Component, error) {
	var (
		ctx     = r.Context()
		err     error
		caseID  = r.PathValue("caseID")
		persons []db.InvestigationTarget
		scenes  []db.InvestigationTarget
	)
	if persons, err = app.queries.ListInvestigationTargets(ctx, db.ListInvestigationTargetsParams{
		CaseID: caseID,
		Type:   "person",
	}); err != nil {
		app.serverError(w, r, err)
		return nil, err
	}
	if scenes, err = app.queries.ListInvestigationTargets(ctx, db.ListInvestigationTargetsParams{
		CaseID: caseID,
		Type:   "scene",
	}); err != nil {
		app.serverError(w, r, err)
		return nil, err
	}
	caseHome := components.CaseHome(persons, scenes)
	return &caseHome, nil
}
