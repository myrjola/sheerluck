package main

import (
	"html/template"
	"log"
	"net/http"
)

const internalServerError = "Internal Server Error"

func home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Use the template.ParseFiles() function to read the template file into a
	// template set. If there's an error, we log the detailed error message and use
	// the http.Error() function to send a generic 500 Internal Server Error
	// response to the user. Note that we use the net/http constant
	// http.StatusInternalServerError here instead of the integer 500 directly.
	ts, err := template.ParseFiles("./ui/html/base.gohtml", "./ui/html/nav.gohtml")
	if err != nil {
		log.Print(err.Error())
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	// Detect htmx header and render partial template
	if r.Header.Get("Hx-Request") == "true" {
		err = ts.ExecuteTemplate(w, "button", nil)
	} else {
		// Then we use the Execute() method on the template set to write the
		// template content as the response body. The last parameter to Execute()
		// represents any dynamic data that we want to pass in, which for now we'll
		// leave as nil.
		err = ts.ExecuteTemplate(w, "base", nil)
	}

	if err != nil {
		log.Print(err.Error())
		http.Error(w, internalServerError, http.StatusInternalServerError)
	}
}

func swap(w http.ResponseWriter, _ *http.Request) {
	// Use the template.ParseFiles() function to read the template file into a
	// template set. If there's an error, we log the detailed error message and use
	// the http.Error() function to send a generic 500 Internal Server Error
	// response to the user. Note that we use the net/http constant
	// http.StatusInternalServerError here instead of the integer 500 directly.
	ts, err := template.ParseFiles("./ui/html/swap.gohtml")
	if err != nil {
		log.Print(err.Error())
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	// Then we use the Execute() method on the template set to write the
	// template content as the response body. The last parameter to Execute()
	// represents any dynamic data that we want to pass in, which for now we'll
	// leave as nil.
	err = ts.Execute(w, nil)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, internalServerError, http.StatusInternalServerError)
	}
}
