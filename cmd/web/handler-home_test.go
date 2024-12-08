package main

import (
	"github.com/alexedwards/scs/v2"
	"github.com/myrjola/sheerluck/internal/ai"
	"github.com/myrjola/sheerluck/internal/repositories"
	"github.com/myrjola/sheerluck/internal/webauthnhandler"
	"log/slog"
	"net/http"
	"testing"
)

func Test_application_home(t *testing.T) {
	type fields struct {
		logger          *slog.Logger
		aiClient        ai.Client
		webAuthnHandler *webauthnhandler.WebAuthnHandler
		sessionManager  *scs.SessionManager
		investigations  *repositories.InvestigationRepository
	}
	type args struct {
		w http.ResponseWriter
		r *http.Request
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &application{
				logger:          tt.fields.logger,
				aiClient:        tt.fields.aiClient,
				webAuthnHandler: tt.fields.webAuthnHandler,
				sessionManager:  tt.fields.sessionManager,
				investigations:  tt.fields.investigations,
			}
			app.home(tt.args.w, tt.args.r)
		})
	}
}
