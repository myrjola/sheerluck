package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/sashabaranov/go-openai"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
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

func (app *application) resolveRoutes(currentPath string) []route {
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

func (app *application) renderPage(w http.ResponseWriter, r *http.Request, t *template.Template, data any) error {
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

type questionPeopleData struct {
	Routes        []route
	ChatResponses []chatResponse
}

func (app *application) questionPeople(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := app.postQuestion(r)
		if err != nil {
			app.serverError(w, r, err)
			return
		}
	}

	var (
		t   *template.Template
		err error
	)

	if t, err = app.compileTemplates("question-people", "partials/chat-responses"); err != nil {
		app.serverError(w, r, err)
		return
	}

	routes := app.resolveRoutes(r.URL.Path)

	data := questionPeopleData{
		Routes:        routes,
		ChatResponses: chatResponses,
	}

	if err = app.renderPage(w, r, t, data); err != nil {
		app.serverError(w, r, err)
		return
	}
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

	routes := app.resolveRoutes(r.URL.Path)

	data := baseData{
		Routes: routes,
	}

	if err = app.renderPage(w, r, t, data); err != nil {
		app.serverError(w, r, err)
		return
	}
}

type chatResponse struct {
	Question string
	Answer   string
}

var chatResponses = []chatResponse{}

func (app *application) postQuestion(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	question := r.PostForm.Get("question")

	// When HTMX does the request, we defer streaming the data to SSE triggered by the HTMX template.
	if r.Header.Get("Hx-Boosted") == "true" {
		chatResponses = append(chatResponses, chatResponse{
			Question: question,
			Answer:   "",
		})
		return nil
	}

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

	var (
		resp openai.ChatCompletionResponse
		err  error
	)

	if resp, err = app.aiClient.SyncCompletion(messages); err != nil {
		return err
	}

	cr := chatResponse{
		Question: question,
		Answer:   resp.Choices[0].Message.Content,
	}

	chatResponses = append(chatResponses, cr)

	return nil
}

// streamChat sends server side events (SSE) to the client.
func (app *application) streamChat(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Create a channel to send data
	dataCh := make(chan string)

	// Create a context for handling client disconnection
	_, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Send data to the client
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for data := range dataCh {
			app.logger.Debug("Sending data", slog.Any("data", data))
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.(http.Flusher).Flush()
		}
	}()

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

	// HACK: Remove last assistant answer because it's empty at this point
	messages = messages[:len(messages)-1]
	question := messages[len(messages)-1].Content

	stream, err := app.aiClient.StreamCompletion(messages)
	if err != nil {
		app.serverError(w, r, err)
		return
	}
	defer stream.Close()

	cr := chatResponse{
		Question: question,
		Answer:   "",
	}
	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			app.logger.Debug("stream finished")
			dataCh <- "<div id='chat-listener' hx-swap-oob='true'></div>"
			break
		}

		if err != nil {
			app.logger.Error("stream error", slog.Any("err", err))
			break
		}

		delta := response.Choices[0].Delta.Content

		cr.Answer += delta
		dataCh <- fmt.Sprintf("<span>%s</span>", strings.ReplaceAll(delta, "\n", "<br>"))
	}

	chatResponses[len(chatResponses)-1] = cr
	wg.Wait()
}

type User struct {
	DisplayName string
	credential  webauthn.Credential
}

var user = User{}
var session *webauthn.SessionData

// WebAuthnID provides the user handle of the user account. A user handle is an opaque byte sequence with a maximum
// size of 64 bytes, and is not meant to be displayed to the user.
//
// To ensure secure operation, authentication and authorization decisions MUST be made on the basis of this id
// member, not the displayName nor name members. See Section 6.1 of [RFC8266].
//
// It's recommended this value is completely random and uses the entire 64 bytes.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dom-publickeycredentialuserentity-id)
func (u User) WebAuthnID() []byte {
	return []byte{}
}

// WebAuthnName provides the name attribute of the user account during registration and is a human-palatable name for the user
// account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party SHOULD let the user
// choose this, and SHOULD NOT restrict the choice more than necessary.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dictdef-publickeycredentialuserentity)
func (u User) WebAuthnName() string {
	return u.DisplayName
}

// WebAuthnDisplayName provides the name attribute of the user account during registration and is a human-palatable
// name for the user account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party
// SHOULD let the user choose this, and SHOULD NOT restrict the choice more than necessary.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialuserentity-displayname)
func (u User) WebAuthnDisplayName() string {
	return u.DisplayName
}

// WebAuthnCredentials provides the list of Credentials owned by the user.
func (u User) WebAuthnCredentials() []webauthn.Credential {
	return []webauthn.Credential{u.credential}
}

// AddWebAuthnCredentials adds Credential to the user.
func (u User) AddWebAuthnCredential(credential webauthn.Credential) {
	u.credential = credential
}

// WebAuthnIcon is a deprecated option.
// Deprecated: this has been removed from the specification recommendation. Suggest a blank string.
func (u User) WebAuthnIcon() string {
	return ""
}

func (app *application) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	user = User{
		DisplayName: fmt.Sprintf("Anonymous user created at %s", time.Now().Format(time.RFC3339)),
	}

	authSelect := protocol.AuthenticatorSelection{
		AuthenticatorAttachment: protocol.AuthenticatorAttachment("platform"),
		RequireResidentKey:      protocol.ResidentKeyNotRequired(),
		UserVerification:        protocol.VerificationDiscouraged,
	}

	opts, sess, err := app.webAuthn.BeginRegistration(user, webauthn.WithAuthenticatorSelection(authSelect))
	if err != nil {
		app.serverError(w, r, err)
	}

	// store the sessionData values
	session = sess

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(opts)
}

func (app *application) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	credential, err := app.webAuthn.FinishRegistration(user, *session, r)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	user.AddWebAuthnCredential(*credential)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode("Registration Success")
}

func (app *application) BeginLogin(w http.ResponseWriter, r *http.Request) {
	options, sess, err := app.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	session = sess

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}
func findUserHandler(rawID, userHandle []byte) (user webauthn.User, err error) {
	if !bytes.Equal(userHandle, user.WebAuthnID()) {
		return user, nil
	}
	return User{}, errors.New("no user found")
}

func (app *application) FinishLogin(w http.ResponseWriter, r *http.Request) {
	credential, err := app.webAuthn.FinishDiscoverableLogin(findUserHandler, *session, r)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// If login was successful, update the credential object
	// Pseudocode to update the user credential.
	user.AddWebAuthnCredential(*credential)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode("Login Success")
}
