package main

import (
	"context"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"html/template"
	"log"
	"net/http"
	"os"
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

func renderPage(w http.ResponseWriter, r *http.Request, t *template.Template, data any) {
	var err error
	// Detect htmx header and render only the body because that's what's replaced with hx-boost="true"
	if r.Header.Get("Hx-Boosted") == "true" {
		err = t.ExecuteTemplate(w, "body", data)
	} else {
		err = t.ExecuteTemplate(w, "base", data)
	}

	if err != nil {
		log.Print(err.Error())
		http.Error(w, internalServerError, http.StatusInternalServerError)
	}
}

type questionPeopleData struct {
	Routes        []route
	ChatResponses []chatResponse
}

func questionPeople(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := postQuestion(r)
		if err != nil {
			log.Print(err.Error())
			http.Error(w, internalServerError, http.StatusInternalServerError)
			return
		}
	}

	t, err := compileTemplates("question-people", "partials/chat-responses")

	if err != nil {
		log.Print(err.Error())
		http.Error(w, internalServerError, http.StatusInternalServerError)
		return
	}

	routes := resolveRoutes(r.URL.Path)

	data := questionPeopleData{
		Routes:        routes,
		ChatResponses: chatResponses,
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

type chatResponse struct {
	Question string
	Answer   string
}

var chatResponses = []chatResponse{}

func postQuestion(r *http.Request) error {
	err := r.ParseForm()
	if err != nil {
		return err
	}
	question := r.PostForm.Get("question")

	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

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

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
		},
	)

	if err != nil {
		log.Print(err.Error())
		return err
	}

	cr := chatResponse{
		Question: question,
		Answer:   resp.Choices[0].Message.Content,
	}
	chatResponses = append(chatResponses, cr)

	return nil
}
