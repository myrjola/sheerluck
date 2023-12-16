package main

import (
	"github.com/justinas/alice"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	session := alice.New(app.sessionManager.LoadAndSave)

	mux.Handle("/question-people", session.ThenFunc(app.questionPeople))
	mux.Handle("/question-people/stream", session.ThenFunc(app.streamChat))
	mux.Handle("/investigate-scenes", session.ThenFunc(app.investigateScenes))

	mux.Handle("/api/registration/start", session.ThenFunc(app.BeginRegistration))
	mux.Handle("/api/registration/finish", session.ThenFunc(app.FinishRegistration))
	mux.Handle("/api/login/start", session.ThenFunc(app.BeginLogin))
	mux.Handle("/api/login/finish", session.ThenFunc(app.FinishLogin))

	return app.recoverPanic(app.logRequest(secureHeaders(mux)))
}
