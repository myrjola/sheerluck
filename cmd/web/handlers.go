package main

import (
	"bytes"
	"context"
	"encoding/gob"
	"fmt"
	"github.com/a-h/templ"
	"github.com/donseba/go-htmx"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/myrjola/sheerluck/internal/contexthelpers"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/ui/components"
	"github.com/sashabaranov/go-openai"
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

func init() {
	gob.Register(webauthn.SessionData{})
}

type slotFunc func(w http.ResponseWriter, r *http.Request, h *htmx.HxRequestHeader) (*templ.Component, error)

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

func (app *application) htmxHandler(slotF slotFunc) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		var (
			err  error
			slot *templ.Component
		)
		ctx := r.Context()
		handler := app.htmx.NewHandler(w, r)
		headers := handler.Request()

		nav := components.Nav()
		if slot, err = slotF(w, r, &headers); err != nil {
			// We assume the slot function already responded with error code
			return
		}
		body := components.Body(*slot, nav)

		if headers.HxRequest {
			err = body.Render(ctx, w)
		} else {
			err = components.Base(body).Render(ctx, w)
		}

		if err != nil {
			app.serverError(w, r, err)
			return
		}
	}

	return http.HandlerFunc(fn)
}

func (app *application) startCompletionStream(ctx context.Context, completionID uuid.UUID, messages []openai.ChatCompletionMessage, question string) error {
	logger := app.logger.With("completionID", completionID)
	authenticatedUserID := contexthelpers.AuthenticatedUserID(ctx)

	completionChan := make(chan struct {
		string
		error
	})
	app.broker.Publish(completionID, completionChan)
	go func() {
		answer := ""
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
			app.broker.Unpublish(completionID)
			close(completionChan)
		}()
		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				logger.Debug("stream finished")
				if err := app.investigations.FinishCompletion(
					context.Background(),
					"le-bon",
					authenticatedUserID,
					question,
					answer,
				); err != nil {
					app.logger.Error("finish completion", slog.Any("error", err))
					return
				}
				logger.Debug("completion persisted")

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
			answer += delta

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
		err                   error
		chatResponses         []components.ChatResponse
		ctx                   = r.Context()
		investigationTargetID = "le-bon"
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
	completionID := uuid.New()

	handler := app.htmx.NewHandler(w, r)
	headers := handler.Request()

	investigation, err := app.investigations.Get(ctx, investigationTargetID, contexthelpers.AuthenticatedUserID(ctx))

	messages := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: "This is a murder mystery game based on Murders in the Rue Morgue by Edgar Allan Poe. " +
				"You are Adolphe Le Bon, a clerk who was arrested based on circumstantial evidence on the murder of " +
				"Madame L'Espanaye and her daughter. Answer the questions from detective Auguste Dupin in plain text.",
		},
	}

	for _, completion := range investigation.Completions {
		chatResponses = append(chatResponses, components.ChatResponse{
			Question: completion.Question,
			Answer:   completion.Answer,
		})
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
		if err = app.startCompletionStream(ctx, completionID, messages, question); err != nil {
			app.serverError(w, r, fmt.Errorf("start completion stream: %w", err))
			return
		}
		cr := components.ChatResponse{
			Question:     question,
			Answer:       "",
			CompletionID: completionID.String(),
		}
		app.logger.Info("completion stream started", slog.Any("completionID", completionID))
		if err := components.ActiveChatResponse(cr).Render(r.Context(), w); err != nil {
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
	cr := components.ChatResponse{
		Question: question,
		Answer:   resp.Choices[0].Message.Content,
	}

	if err := app.investigations.FinishCompletion(ctx, "le-bon", contexthelpers.AuthenticatedUserID(ctx), cr.Question, cr.Answer); err != nil {
		app.serverError(w, r, err)
		return
	}

	http.Redirect(w, r, "/question-people", http.StatusSeeOther)
}

// streamChat sends server side events (SSE) to the client.
func (app *application) streamChat(w http.ResponseWriter, r *http.Request) {
	var (
		completionID uuid.UUID
		err          error
	)

	if completionID, err = uuid.Parse(r.PathValue("completionID")); err != nil {
		app.serverError(w, r, err)
		return
	}

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

		sseChannel <- fmt.Sprintf("<span>%s</span>", strings.ReplaceAll(delta, "\n", "<br>"))
	}

	// instruct client to stop listening to SSE stream
	sseChannel <- fmt.Sprintf("<div id='chat-listener-%s' hx-swap-oob='true'></div>", completionID.String())
	close(sseChannel)

	wg.Wait()
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
