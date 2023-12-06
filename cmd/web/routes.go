package main

import "net/http"

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("/question-people", app.questionPeople)
	mux.HandleFunc("/question-people/stream", app.streamChat)
	mux.HandleFunc("/investigate-scenes", app.investigateScenes)

	mux.HandleFunc("/api/registration/start", app.BeginRegistration)
	mux.HandleFunc("/api/registration/finish", app.FinishRegistration)
	mux.HandleFunc("/api/login/start", app.BeginLogin)
	mux.HandleFunc("/api/login/finish", app.FinishLogin)

	return app.recoverPanic(app.logRequest(secureHeaders(mux)))
}
