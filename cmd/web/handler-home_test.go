package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/descope/virtualwebauthn"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	url2 "net/url"
	"os"
	"strings"
	"testing"
	"time"
)

// waitForReady calls the specified endpoint until it gets a HTTP 200 Success
// response or until the context is cancelled or the 1-second timeout is reached.
func waitForReady(ctx context.Context, endpoint string) error {
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
	ctx, cancel := context.WithCancel(context.Background())

	// We need to grab the dynamically allocated port from the log output.
	addrCh := make(chan string, 1)
	logger := slog.New(logging.NewContextHandler(slog.NewTextHandler(w, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == "addr" {
				addrCh <- a.Value.String()
			}
			return a
		},
	})))

	// Start the server and wait for it to be ready.
	go func() {
		if err := run(ctx, logger, lookupEnv); err != nil {
			cancel()
			assert.NoError(t, err)
		}
	}()
	select {
	case <-ctx.Done():
		t.Fatal("server failed to start")
		return ""
	case addr := <-addrCh:
		// swap 127.0.0.1 with localhost to make secure cookies work in [cookiejar.Jar]
		port := strings.Split(addr, ":")[1]
		serverURL := fmt.Sprintf("http://localhost:%s", port)
		if err := waitForReady(ctx, fmt.Sprintf("%s/api/healthy", serverURL)); err != nil {
			require.NoError(t, err)
		}
		return serverURL
	}
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

	require.Equal(t, 1, doc.Find("button:contains('Sign in')").Length())
	require.Equal(t, 1, doc.Find("button:contains('Register')").Length())

	rp := virtualwebauthn.RelyingParty{Name: "Sheerluck", ID: "localhost", Origin: "http://localhost:0"}
	authenticator := virtualwebauthn.NewAuthenticator()
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

	// Ask the server to start a register flow for a user. The server and user here
	// are placeholders for whatever the system being tested uses.
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{Jar: jar}
	resp, err := client.Post(url+"/api/registration/start", "application/json", nil)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Print out headers
	fmt.Printf("headers: %v+", resp.Header)

	bodyBytes, err := io.ReadAll(resp.Body)
	err = resp.Body.Close()
	fmt.Printf("start: %s", bodyBytes)
	require.NoError(t, err)
	parsedAttestationOptions, err := virtualwebauthn.ParseAttestationOptions(string(bodyBytes))
	require.NoError(t, err)

	// Creates an attestation response that we can send to the relying party as if it came from
	// an actual browser and authenticator.
	attestationResponse := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *parsedAttestationOptions)

	// Finish the register flow by sending the attestation response. Again the server and
	// user here are placeholders for whatever the system being tested uses.
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(attestationResponse)
	require.NoError(t, err)
	parsedUrl, err := url2.Parse(url)
	require.NoError(t, err)
	fmt.Printf("cookie jar: %v+", client.Jar)
	cookies := client.Jar.Cookies(parsedUrl)
	fmt.Printf("cookies: %v+", cookies)
	fmt.Printf("buf: %s", buf.String())
	resp, err = client.Post(url+"/api/registration/finish", "application/json", strings.NewReader(attestationResponse))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// at this point, our credential is ready for logging in
	authenticator.AddCredential(credential)

	res, err = client.Get(url)
	require.NoError(t, err)
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		assert.NoError(t, err)
	}(res.Body)

	require.Equal(t, http.StatusOK, res.StatusCode)

	doc, err = goquery.NewDocumentFromReader(res.Body)
	require.NoError(t, err)
	require.Equal(t, 1, doc.Find("button:contains('Log out')").Length())
	// TODOs for nicer test setup:
	// - http client with
	//     - cookie jar
	//     - json endcode/decode support
	//     - html assertion helper
	//     - form submission helper
	// - webauthn glue for setting up users for tests

}
