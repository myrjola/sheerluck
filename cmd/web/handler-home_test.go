package main

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"
)

// waitForReady calls the specified endpoint until it gets a HTTP 200 Success
// response or until the context is cancelled or the 1-second timeout is reached.
func waitForReady(
	ctx context.Context,
	endpoint string,
) error {
	timeout := 1 * time.Second
	client := http.Client{}
	startTime := time.Now()
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)
	for {
		if req, err = http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			endpoint,
			nil,
		); err != nil {
			return errors.Wrap(err, "create request")
		}

		if resp, err = client.Do(req); err == nil {
			if resp.StatusCode == http.StatusOK {
				if err = resp.Body.Close(); err != nil {
					return errors.Wrap(err, "close response body")
				}
				return nil
			}
			if err = resp.Body.Close(); err != nil {
				return errors.Wrap(err, "close response body")
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if time.Since(startTime) >= timeout {
				return errors.New("timeout waiting for endpoint to be ready")
			}
			time.Sleep(250 * time.Millisecond)
		}
	}
}

func testLookupEnv(key string) (string, bool) {
	switch key {
	case "SHEERLUCK_ADDR":
		return "localhost:0", true
	default:
		return "", false
	}
}

// startTestServer starts the test server, waits for it to be ready, and return the server URL for testing.
func startTestServer(t *testing.T, w io.Writer, lookupEnv func(string) (string, bool)) string {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	// We need to grab the dynamically allocated port from the log output.
	addrCh := make(chan string, 1)
	logger := slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == "addr" {
				addrCh <- a.Value.String()
			}
			return a
		},
	}))

	// Start the server and wait for it to be ready.
	go func() {
		err := run(ctx, logger, lookupEnv)
		assert.NoError(t, err)
	}()
	addr := <-addrCh
	serverURL := fmt.Sprintf("http://%s", addr)
	if err := waitForReady(ctx, fmt.Sprintf("%s/api/healthy", serverURL)); err != nil {
		require.NoError(t, err)
	}
	return serverURL
}

func Test_application_home(t *testing.T) {
	url := startTestServer(t, os.Stdout, testLookupEnv)

	res, err := http.Get(url)
	require.NoError(t, err)
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		assert.NoError(t, err)
	}(res.Body)

	require.Equal(t, http.StatusOK, res.StatusCode)

	doc, err := goquery.NewDocumentFromReader(res.Body)
	require.NoError(t, err)

	// Find the review items
	doc.Find("button").Each(func(i int, s *goquery.Selection) {
		title := s.Text()
		fmt.Printf("Button %d: %s\n", i, title)
	})
}
