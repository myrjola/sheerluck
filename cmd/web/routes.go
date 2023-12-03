package main

import "net/http"

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	mux.HandleFunc("/question-people", app.questionPeople)
	mux.HandleFunc("/question-people/stream", app.streamChat)
	mux.HandleFunc("/investigate-scenes", app.investigateScenes)

	return mux
}
