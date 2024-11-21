package main

import (
	"net/http"
)

type homeTemplateData struct {
	BaseTemplateData
}

func (app *application) home(w http.ResponseWriter, r *http.Request) {
	data := homeTemplateData{
		BaseTemplateData: newBaseTemplateData(r),
	}

	// Render the home page.
	app.render(w, r, http.StatusOK, "page-home.gohtml", data)
}
