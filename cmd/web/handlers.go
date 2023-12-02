package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

const internalServerError = "Internal Server Error"

type route struct {
	Href    string
	Title   string
	Current bool
}

type baseData struct {
	Routes []route
}

func resolveRoutes(currentPath string) []route {
	routes := []route{
		{
			Href:  "/question-people",
			Title: "Question people",
		},
		{
			Href:  "/investigate-scenes",
			Title: "Investigate scenes",
		},
	}

	for i := range routes {
		routes[i].Current = currentPath == routes[i].Href
	}
	return routes
}

// compileTemplates parses the base templates and adds a templates based on path
func compileTemplates(templateFileNames ...string) (*template.Template, error) {
	templates := []string{
		"./ui/html/base.gohtml",
		"./ui/html/nav/nav.gohtml",
	}

	for _, templateFilename := range templateFileNames {
		templates = append(templates, fmt.Sprintf("./ui/html/%s.gohtml", templateFilename))
	}

	return template.ParseFiles(templates...)
}

func renderPage(w http.ResponseWriter, r *http.Request, template *template.Template, data any) {
	var err error
	// Detect htmx header and render only the body because that's what's replaced with hx-boost="true"
	if r.Header.Get("Hx-Boosted") == "true" {
		err = template.ExecuteTemplate(w, "body", data)
	} else {
		err = template.ExecuteTemplate(w, "base", data)
	}

	if err != nil {
		log.Print(err.Error())
		http.Error(w, internalServerError, http.StatusInternalServerError)
	}
}

func questionPeople(w http.ResponseWriter, r *http.Request) {
	t, err := compileTemplates("question-people")

	if err != nil {
		log.Print(err.Error())
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	routes := resolveRoutes(r.URL.Path)

	data := baseData{
		Routes: routes,
	}

	renderPage(w, r, t, data)
}

func investigateScenes(w http.ResponseWriter, r *http.Request) {
	t, err := compileTemplates("investigate-scenes")

	if err != nil {
		log.Print(err.Error())
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	routes := resolveRoutes(r.URL.Path)

	data := baseData{
		Routes: routes,
	}

	renderPage(w, r, t, data)
}
