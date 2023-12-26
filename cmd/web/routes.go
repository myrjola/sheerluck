package main

import (
	"github.com/justinas/alice"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/", cacheForeverHeaders(fileServer))

	session := alice.New(app.sessionManager.LoadAndSave, app.authenticate, noSurf)
	sessionSSE := alice.New(app.serverSentEventMiddleware, app.authenticate)

	mux.Handle("GET /{$}", session.ThenFunc(app.Home))
	mux.Handle("/question-people", session.ThenFunc(app.questionPeople))
	mux.Handle("GET /question-people/stream", sessionSSE.ThenFunc(app.streamChat))
	mux.Handle("GET /investigate-scenes", session.ThenFunc(app.investigateScenes))

	mux.Handle("POST /api/registration/start", session.ThenFunc(app.BeginRegistration))
	mux.Handle("POST /api/registration/finish", session.ThenFunc(app.FinishRegistration))
	mux.Handle("POST /api/login/start", session.ThenFunc(app.BeginLogin))
	mux.Handle("POST /api/login/finish", session.ThenFunc(app.FinishLogin))
	mux.Handle("POST /api/logout", session.ThenFunc(app.Logout))

	common := alice.New(app.recoverPanic, app.logRequest, secureHeaders)

	return common.Then(mux)
}
