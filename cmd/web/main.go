package main

import (
	"flag"
	"github.com/joho/godotenv"
	"log"
	"log/slog"
	"net/http"
	"os"
)

func main() {
	addr := flag.String("addr", ":4000", "HTTP network address")
	flag.Parse()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	loggerHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	})
	logger := slog.New(loggerHandler)

	// Use the http.NewServeMux() function to initialize a new servemux, then
	// register the questionPeople function as the handler for the "/" URL pattern.
	mux := http.NewServeMux()
	mux.HandleFunc("/question-people", questionPeople)
	mux.HandleFunc("/question-people/stream", streamChat)
	mux.HandleFunc("/investigate-scenes", investigateScenes)

	// Create a file server which serves files out of the "./ui/static" directory.
	// Note that the Href given to the http.Dir function is relative to the project
	// directory root.
	fileServer := http.FileServer(http.Dir("./ui/static/"))

	// Use the mux.Handle() function to register the file server as the handler for
	// all URL paths that start with "/static/". For matching paths, we strip the
	// "/static" prefix before the request reaches the file server.
	mux.Handle("/static/", http.StripPrefix("/static", fileServer))

	// Print a log a message to say that the server is starting.
	logger.Info("starting server", slog.Any("addr", ":4000"))

	// Use the http.ListenAndServe() function to start a new web server. We pass in
	// two parameters: the TCP network address to listen on (in this case ":4000")
	// and the servemux we just created. If http.ListenAndServe() returns an error
	// we use the log.Fatal() function to log the error message and exit. Note
	// that any error returned by http.ListenAndServe() is always non-nil.
	err = http.ListenAndServe(*addr, mux)
	logger.Error(err.Error())
	os.Exit(1)
}
