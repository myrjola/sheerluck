package main

import (
	"github.com/justinas/alice"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static/"))
	mux.Handle("/", cacheForeverHeaders(fileServer))

	session := alice.New(app.sessionManager.LoadAndSave, app.webAuthnHandler.AuthenticateMiddleware)
	mustSession := alice.New(
		app.sessionManager.LoadAndSave,
		app.webAuthnHandler.AuthenticateMiddleware,
		app.mustAuthenticate,
	)

	mux.Handle("GET /{$}", session.ThenFunc(app.home))
	mux.Handle("GET /test", session.ThenFunc(app.home))
	mux.Handle("GET /cases/{caseID}/investigation-targets/{investigationTargetID}",
		mustSession.ThenFunc(app.investigateTargetGET))
	mux.Handle("POST /cases/{caseID}/investigation-targets/{investigationTargetID}",
		mustSession.ThenFunc(app.investigateTargetPOST))

	mux.Handle("POST /api/registration/start", session.ThenFunc(app.beginRegistration))
	mux.Handle("POST /api/registration/finish", session.ThenFunc(app.finishRegistration))
	mux.Handle("POST /api/login/start", session.ThenFunc(app.beginLogin))
	mux.Handle("POST /api/login/finish", session.ThenFunc(app.finishLogin))
	mux.Handle("POST /api/logout", session.ThenFunc(app.logout))

	mux.Handle("GET /api/healthy", session.ThenFunc(app.healthy))

	common := alice.New(app.recoverPanic, app.logRequest, secureHeaders, noSurf, commonContext)

	return common.Then(mux)
}
