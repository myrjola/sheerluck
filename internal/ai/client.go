package ai

import (
	"context"
	"github.com/sashabaranov/go-openai"
	"os"
)

type Client struct {
	client *openai.Client
}

func NewClient() Client {
	return Client{
		client: openai.NewClient(os.Getenv("OPENAI_API_KEY")),
	}
}

func (c *Client) SyncCompletion(messages []openai.ChatCompletionMessage) (openai.ChatCompletionResponse, error) {
	return c.client.CreateChatCompletion(
		context.TODO(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
		},
	)
}

func (c *Client) StreamCompletion(messages []openai.ChatCompletionMessage) (*openai.ChatCompletionStream, error) {
	return c.client.CreateChatCompletionStream(
		context.TODO(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: messages,
		},
	)
}
