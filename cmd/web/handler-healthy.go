package main

import "net/http"

// healthy responds with a JSON object indicating that the server is healthy.
func (app *application) healthy(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}
