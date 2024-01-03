package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/donseba/go-htmx"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/myrjola/sheerluck/internal/models"
	"github.com/sashabaranov/go-openai"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
)

type route struct {
	Href    string
	Title   string
	Current bool
	Icon    string
}

func init() {
	gob.Register(webauthn.SessionData{})
}

func (app *application) parseTemplates() (*template.Template, error) {
	fs := os.DirFS("./ui/html")
	patterns := []string{
		"base.gohtml",
		"partials/*.gohtml",
		"pages/*.gohtml",
	}

	return template.New("base").Funcs(templateFuncMap).ParseFS(fs, patterns...)
}

// compileTemplates parses the base templates and adds a templates based on path
func (app *application) compileTemplates(templateFileNames ...string) (*template.Template, error) {
	templates := []string{
		"./ui/html/base.gohtml",
		"./ui/html/nav/nav.gohtml",
	}

	for _, templateFilename := range templateFileNames {
		templates = append(templates, fmt.Sprintf("./ui/html/%s.gohtml", templateFilename))
	}

	return template.ParseFiles(templates...)
}

func (app *application) htmxHandler(slotFunc func(ctx context.Context, w io.Writer, h *htmx.HxRequestHeader) error) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		handler := app.htmx.NewHandler(w, r)
		headers := handler.Request()

		nav := bytes.Buffer{}
		if err := app.Nav(ctx, &nav, &headers); err != nil {
			app.serverError(w, r, err)
			return
		}
		slot := bytes.Buffer{}
		if err := slotFunc(ctx, &slot, &headers); err != nil {
			app.serverError(w, r, err)
			return
		}
		data := baseData{
			Nav:  template.HTML(nav.String()),  //nolint:gosec
			Slot: template.HTML(slot.String()), //nolint:gosec
			Ctx:  ctx,
		}

		templateName := "base"

		if headers.HxRequest {
			templateName = "body"
		}

		if err := app.executeTemplate(w, templateName, data); err != nil {
			app.serverError(w, r, err)
			return
		}
	}

	return http.HandlerFunc(fn)
}

func (app *application) renderHtmx(w http.ResponseWriter, r *http.Request, t *template.Template, data any) error {
	var err error
	// Detect htmx header and render only the body because that's what's replaced with hx-boost="true"
	if r.Header.Get("Hx-Boosted") == "true" {
		err = t.ExecuteTemplate(w, "body", data)
	} else {
		err = t.ExecuteTemplate(w, "base", data)
	}

	if err != nil {
		app.serverError(w, r, err)
		return err
	}

	return nil
}

func (app *application) investigateScenes(w http.ResponseWriter, r *http.Request) {
	var (
		t   *template.Template
		err error
	)
	if t, err = app.compileTemplates("investigate-scenes"); err != nil {
		app.serverError(w, r, err)
		return
	}

	//routes := app.resolveRoutes(r.URL.Path)

	data := baseData{
		//IsAuthenticated: contexthelpers.IsAuthenticated(r.Context()),
		//Routes:          routes,
	}

	if err = app.renderHtmx(w, r, t, data); err != nil {
		app.serverError(w, r, err)
		return
	}
}

type chatResponse struct {
	Question string
	Answer   string
}

var chatResponses []chatResponse

func (app *application) startCompletionStream(completionID int, messages []openai.ChatCompletionMessage) error {
	logger := app.logger.With("completionID", completionID)

	completionChan := make(chan struct {
		string
		error
	})
	app.broker.Publish(1, completionChan)
	go func() {
		stream, err := app.aiClient.StreamCompletion(messages)
		if err != nil {
			app.logger.Error("completion stream: %w", err)
			return
		}
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error("completion stream: %w", err)
			}
			stream.Close()
			app.broker.Unpublish(1)
			close(completionChan)
		}()
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				logger.Debug("stream finished")
				break
			}

			if err != nil {
				logger.Error("stream error", slog.Any("err", err))
				completionChan <- struct {
					string
					error
				}{"", err}
				break
			}

			delta := response.Choices[0].Delta.Content

			completionChan <- struct {
				string
				error
			}{delta, err}
		}
	}()

	return nil
}

func (app *application) questionTarget(w http.ResponseWriter, r *http.Request) {
	var (
		err error
	)
	if err = r.ParseForm(); err != nil {
		app.serverError(w, r, err)
		return
	}
	if r.PostForm == nil {
		app.serverError(w, r, errors.New("no post form"))
		return
	}
	question := r.PostForm.Get("question")
	if question == "" {
		app.serverError(w, r, errors.New("no question"))
		return
	}
	completionID := 1

	handler := app.htmx.NewHandler(w, r)
	headers := handler.Request()

	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: "This is a murder mystery game based on Murders in the Rue Morgue by Edgar Allan Poe. " +
				"You are Adolphe Le Bon, a clerk who was arrested based on circumstantial evidence on the murder of " +
				"Madame L'Espanaye and her daughter. Answer the questions from detective Auguste Dupin in plain text.",
		},
	}

	for _, response := range chatResponses {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: response.Question,
		})
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: response.Answer,
		})
	}

	messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: question})

	// When HTMX does the request, we start a stream that the SSE GET can listen to through app.broker.
	if headers.HxBoosted {
		if err = app.startCompletionStream(completionID, messages); err != nil {
			app.serverError(w, r, fmt.Errorf("start completion stream: %w", err))
			return
		}
		cr := chatResponse{
			Question: question,
			Answer:   "",
		}
		chatResponses = append(chatResponses, cr)
		if err := app.ChatResponse(w, cr); err != nil {
			app.serverError(w, r, fmt.Errorf("render chat response: %w", err))
		}
		return
	}
	// When not a HTMX request, we do the completion synchronously.
	var resp openai.ChatCompletionResponse
	if resp, err = app.aiClient.SyncCompletion(messages); err != nil {
		app.serverError(w, r, fmt.Errorf("sync completion: %w", err))
		return
	}
	cr := chatResponse{
		Question: question,
		Answer:   resp.Choices[0].Message.Content,
	}
	chatResponses = append(chatResponses, cr)

	http.Redirect(w, r, "/question-people", http.StatusSeeOther)
}

// streamChat sends server side events (SSE) to the client.
func (app *application) streamChat(w http.ResponseWriter, r *http.Request) {
	completionID := 1

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sseChannel := make(chan string)

	// Create a context for handling client disconnection
	_, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Send data to the client
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer func() {
			if err := recover(); err != nil {
				app.logger.Error("send SSE data: %w", err)
			}
			wg.Done()
		}()
		for data := range sseChannel {
			app.logger.Debug("Sending data", slog.Any("data", data))
			_, err := fmt.Fprintf(w, "data: %s\n\n", data)
			if err != nil {
				app.logger.Error("send SSE data", slog.Any("err", err))
				return
			}
			w.(http.Flusher).Flush()
		}
	}()

	cr := chatResponses[len(chatResponses)-1]

	completionChan, ok := <-app.broker.Subscribe(completionID)
	if !ok {
		// TODO: rerender page
		sseChannel <- `<div class="text-red500">Refresh page and try again</div>`
		close(sseChannel)
		return
	}

	for payload := range completionChan {
		delta := payload.string
		err := payload.error
		if err != nil {
			app.logger.Error("completion stream error", slog.Any("err", err))
			// TODO: rerender page with error
			sseChannel <- `<div class="text-red500">Error during streaming. Refresh page and try again</div>`
			break
		}

		cr.Answer += delta
		sseChannel <- fmt.Sprintf("<span>%s</span>", strings.ReplaceAll(delta, "\n", "<br>"))
	}

	// instruct client to stop listening to SSE stream
	sseChannel <- "<div id='chat-listener' hx-swap-oob='true'></div>"
	close(sseChannel)

	chatResponses[len(chatResponses)-1] = cr
	wg.Wait()
}

func (app *application) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	var (
		user *models.User
		err  error
		ctx  = r.Context()
	)
	if user, err = models.NewUser(); err != nil {
		app.serverError(w, r, err)
		return
	}

	authSelect := protocol.AuthenticatorSelection{
		//AuthenticatorAttachment: protocol.AuthenticatorAttachment("platform"),
		RequireResidentKey: protocol.ResidentKeyNotRequired(),
		UserVerification:   protocol.VerificationDiscouraged,
	}

	opts, session, err := app.webAuthn.BeginRegistration(
		user,
		webauthn.WithAuthenticatorSelection(authSelect),
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired))
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(ctx, string(webAuthnSessionKey), *session)
	if err = app.users.Upsert(ctx, user); err != nil {
		app.serverError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(opts)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) parseWebauthnSession(r *http.Request) (webauthn.SessionData, error) {
	var (
		session webauthn.SessionData
		ok      bool
		err     error
	)
	if session, ok = app.sessionManager.Get(r.Context(), string(webAuthnSessionKey)).(webauthn.SessionData); !ok {
		err = errors.New("could not parse webauthn.SessionData")
	}
	return session, err
}

func (app *application) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	var (
		session    webauthn.SessionData
		credential *webauthn.Credential
		user       *models.User
		err        error
		ctx        = r.Context()
	)

	if session, err = app.parseWebauthnSession(r); err != nil {
		app.serverError(w, r, err)
		return
	}

	if user, err = app.users.Get(session.UserID); err != nil {
		app.serverError(w, r, err)
		return
	}

	credential, err = app.webAuthn.FinishRegistration(user, session, r)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	user.AddWebAuthnCredential(*credential)
	if err = app.users.Upsert(ctx, user); err != nil {
		app.serverError(w, r, err)
		return
	}

	// Log in the newly registered user
	if err = app.sessionManager.RenewToken(r.Context()); err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), string(userIDSessionKey), user.ID)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode("Registration Success")
	if err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) BeginLogin(w http.ResponseWriter, r *http.Request) {
	options, session, err := app.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	app.sessionManager.Put(r.Context(), string(webAuthnSessionKey), *session)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(options)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) findUserHandler(_, userHandle []byte) (webauthn.User, error) {
	return app.users.Get(userHandle)
}

func (app *application) FinishLogin(w http.ResponseWriter, r *http.Request) {
	var (
		session webauthn.SessionData
		err     error
		user    *models.User
	)
	if session, err = app.parseWebauthnSession(r); err != nil {
		app.serverError(w, r, err)
		return
	}

	// Validate login

	parsedResponse, err := protocol.ParseCredentialRequestResponse(r)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	credential, err := app.webAuthn.ValidateDiscoverableLogin(app.findUserHandler, session, parsedResponse)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// If login was successful, update the credential object
	if user, err = app.users.Get(parsedResponse.Response.UserHandle); err != nil {
		app.serverError(w, r, err)
		return
	}
	// TODO: would be good to do additional security check on signCount and cloneWarning.
	user.AddWebAuthnCredential(*credential)

	if err = app.users.Upsert(r.Context(), user); err != nil {
		app.serverError(w, r, err)
		return
	}

	// Set userID in session
	if err = app.sessionManager.RenewToken(r.Context()); err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Put(r.Context(), string(userIDSessionKey), user.ID)

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode("Login Success")
	if err != nil {
		app.serverError(w, r, err)
		return
	}
}

func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	if err := app.sessionManager.RenewToken(r.Context()); err != nil {
		app.serverError(w, r, err)
		return
	}
	app.sessionManager.Remove(r.Context(), string(userIDSessionKey))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
