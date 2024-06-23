package main

import (
	goHTMX "github.com/donseba/go-htmx/middleware"
	"github.com/justinas/alice"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/", cacheForeverHeaders(fileServer))

	session := alice.New(app.sessionManager.LoadAndSave, app.authenticate)
	sessionSSE := alice.New(app.serverSentEventMiddleware, app.authenticate)

	mux.Handle("GET /{$}", session.Then(app.htmxHandler(app.Home)))
	mux.Handle("GET /question-people", session.Then(app.htmxHandler(app.QuestionPeople)))
	mux.Handle("POST /question-target", session.ThenFunc(app.questionTarget))
	mux.Handle("GET /completions/stream/{completionID}", sessionSSE.ThenFunc(app.streamChat))
	mux.Handle("GET /investigate-scenes", session.Then(app.htmxHandler(app.InvestigateScenes)))
	mux.Handle("GET /cases/{caseID}/{$}", session.Then(app.htmxHandler(app.CaseView)))
	mux.Handle("GET /cases/{caseID}/investigation-targets/{investigationTargetID}/{$}", session.Then(app.htmxHandler(app.QuestionPeople)))

	mux.Handle("POST /api/registration/start", session.ThenFunc(app.BeginRegistration))
	mux.Handle("POST /api/registration/finish", session.ThenFunc(app.FinishRegistration))
	mux.Handle("POST /api/login/start", session.ThenFunc(app.BeginLogin))
	mux.Handle("POST /api/login/finish", session.ThenFunc(app.FinishLogin))
	mux.Handle("POST /api/logout", session.ThenFunc(app.Logout))

	common := alice.New(app.recoverPanic, app.logRequest, secureHeaders, goHTMX.MiddleWare, noSurf, commonContext)

	return common.Then(mux)
}
