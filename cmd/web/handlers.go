package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"github.com/myrjola/sheerluck/internal/errors"
	"html/template"
	"log/slog"
	"net/http"
	"path/filepath"
)

func init() {
	gob.Register(webauthn.SessionData{})
}

// pageTemplate returns a template for the given page name.
//
// pageName corresponds to directory inside ui/templates/pages folder. It has to include a template named "page".
func (app *application) pageTemplate(pageName string) (*template.Template, error) {
	files := []string{
		"ui/templates/base.gohtml",
	}

	pageTemplateFiles, err := filepath.Glob(fmt.Sprintf("ui/templates/pages/%s/*.gohtml", pageName))
	if err != nil {
		return nil, fmt.Errorf("glob page template files: %w", err)
	}
	files = append(files, pageTemplateFiles...)

	// We need to initialize the FuncMap before parsing the files. These will be overridden in the render function.
	return template.New(pageName).Funcs(template.FuncMap{
		"nonce": func() string {
			panic("not implemented")
		},
		"csrf": func() string {
			panic("not implemented")
		},
	}).ParseFiles(files...)
}

func (app *application) render(w http.ResponseWriter, r *http.Request, status int, file string, data any) {
	var (
		err error
		t   *template.Template
	)

	if t, err = app.pageTemplate(file); err != nil {
		app.serverError(w, r, errors.Wrap(err, "parse template", slog.String("template", file)))
		return
	}

	buf := new(bytes.Buffer)
	ctx := r.Context()
	nonce := fmt.Sprintf("nonce=\"%s\"", contexthelpers.CSPNonce(ctx))
	csrf := fmt.Sprintf("<input type=\"hidden\" name=\"csrf_token\" value=\"%s\"/>", contexthelpers.CSRFToken(ctx))
	t.Funcs(template.FuncMap{
		"nonce": func() template.HTMLAttr {
			return template.HTMLAttr(nonce) //nolint:gosec, we trust the nonce since it's not provided by user.
		},
		"csrf": func() template.HTML {
			return template.HTML(csrf) //nolint:gosec, we trust the csrf since it's not provided by user.
		},
	})
	if err = t.ExecuteTemplate(buf, "base", data); err != nil {
		app.serverError(w, r, errors.Wrap(err, "execute template", slog.String("template", file)))
		return
	}

	w.WriteHeader(status)

	_, _ = buf.WriteTo(w)
}

func (app *application) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	out, err := app.webAuthnHandler.BeginRegistration(r.Context())
	w.Header().Set("Content-Type", "application/json")
	if _, err = w.Write(out); err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	if err := app.webAuthnHandler.FinishRegistration(r); err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) BeginLogin(w http.ResponseWriter, r *http.Request) {
	out, err := app.webAuthnHandler.BeginLogin(w, r)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(out)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) FinishLogin(w http.ResponseWriter, r *http.Request) {
	if err := app.webAuthnHandler.FinishLogin(r); err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	if err := app.webAuthnHandler.Logout(r.Context()); err != nil {
		app.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
