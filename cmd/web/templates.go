package main

import (
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"net/http"
)

type BaseTemplateData struct {
	Authenticated bool
}

func newBaseTemplateData(r *http.Request) BaseTemplateData {
	return BaseTemplateData{
		Authenticated: contexthelpers.IsAuthenticated(r.Context()),
	}
}
