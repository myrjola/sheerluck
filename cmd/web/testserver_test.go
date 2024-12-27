package main

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/descope/virtualwebauthn"
	"github.com/justinas/nosurf"
	"github.com/myrjola/sheerluck/internal/errors"
	"github.com/myrjola/sheerluck/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"log/slog"
	"net/http"
	url2 "net/url"
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

type testServer struct {
	url           string
	client        http.Client
	rp            virtualwebauthn.RelyingParty
	authenticator virtualwebauthn.Authenticator
}

// startTestServer starts the test server, waits for it to be ready, and return the server URL for testing.
func startTestServer(t *testing.T, w io.Writer, lookupEnv func(string) (string, bool)) testServer {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())

	// We need to grab the dynamically allocated port from the log output.
	addrCh := make(chan string, 1)
	logger := slog.New(logging.NewContextHandler(slog.NewTextHandler(w, &slog.HandlerOptions{
		AddSource: false,
		Level:     slog.LevelDebug,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == "Addr" {
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
		return testServer{} //nolint:exhaustruct // This is unreachable.
	case addr := <-addrCh:
		serverURL := fmt.Sprintf("http://%s", addr)
		if err := waitForReady(ctx, fmt.Sprintf("%s/api/healthy", serverURL)); err != nil {
			require.NoError(t, err)
		}
		jar, err := newUnsafeCookieJar()
		require.NoError(t, err)
		return testServer{
			url:           serverURL,
			client:        http.Client{Jar: jar},
			rp:            virtualwebauthn.RelyingParty{Name: "Sheerluck", ID: "localhost", Origin: "http://localhost:0"},
			authenticator: virtualwebauthn.NewAuthenticator(),
		}
	}
}

func (s *testServer) Client() http.Client {
	return s.client
}

func (s *testServer) URL() string {
	return s.url
}

// Get fetches a URL and returns the response.
func (s *testServer) Get(t *testing.T, urlPath string) *http.Response {
	t.Helper()
	resp, err := s.client.Get(s.url + urlPath)
	require.NoError(t, err)
	return resp
}

// GetDoc fetches a URL and returns a goquery document.
func (s *testServer) GetDoc(t *testing.T, urlPath string) *goquery.Document {
	t.Helper()
	resp := s.Get(t, urlPath)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer func() {
		err := resp.Body.Close()
		require.NoError(t, err)
	}()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	require.NoError(t, err)
	return doc
}

// NewRequest creates a new HTTP request to the server.
func (s *testServer) NewRequest(t *testing.T, method, urlPath string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, s.url+urlPath, body)
	require.NoError(t, err)
	return req
}

// Register registers a new WebAuthn credential with the server and returns the front page document.
func (s *testServer) Register(t *testing.T) *goquery.Document {
	doc := s.GetDoc(t, "/")

	// Extract CSRF token from the form.
	registrationStartURLPath := "/api/registration/start"
	formSelector := fmt.Sprintf("form[action='%s']", registrationStartURLPath)
	form := doc.Find(formSelector)
	csrfToken, ok := form.Find("input[name=csrf_token]").Attr("value")
	require.True(t, ok, "csrf_token not found in form %s", formSelector)

	// Start Webauthn registration.
	req := s.NewRequest(t, http.MethodPost, registrationStartURLPath, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(nosurf.HeaderName, csrfToken)
	resp, err := s.client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var bodyBytes []byte
	bodyBytes, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = resp.Body.Close()
	require.NoError(t, err)
	attOpts, err := virtualwebauthn.ParseAttestationOptions(string(bodyBytes))
	require.NoError(t, err)
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)

	// Finalise Webauthn registration.
	attestationResponse := virtualwebauthn.CreateAttestationResponse(s.rp, s.authenticator, credential, *attOpts)
	req = s.NewRequest(t, http.MethodPost, "/api/registration/finish", strings.NewReader(attestationResponse))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(nosurf.HeaderName, csrfToken)
	resp, err = s.client.Do(req)
	require.NoError(t, err)
	err = resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// At this point, our credential is ready for logging in.
	s.authenticator.AddCredential(credential)
	// This option is needed for making Passkey login work.
	s.authenticator.Options.UserHandle = []byte(attOpts.UserID)

	return s.GetDoc(t, "/")
}

// Login logs in to the server given there is a registered WebAuthn credential and returns the front page document.
func (s *testServer) Login(t *testing.T) *goquery.Document {
	// Extract CSRF token from the form
	loginStartURLPath := "/api/login/start"
	formSelector := fmt.Sprintf("form[action='%s']", loginStartURLPath)
	doc := s.GetDoc(t, "/")
	form := doc.Find(formSelector)
	csrfToken, ok := form.Find("input[name=csrf_token]").Attr("value")
	require.True(t, ok, "csrf_token not found in form %s", formSelector)

	req, err := http.NewRequest(http.MethodPost, s.url+loginStartURLPath, nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(nosurf.HeaderName, csrfToken)
	var resp *http.Response
	resp, err = s.client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var bodyBytes []byte
	bodyBytes, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	err = resp.Body.Close()
	require.NoError(t, err)
	asOpts, err := virtualwebauthn.ParseAssertionOptions(string(bodyBytes))
	require.NoError(t, err)
	credential := s.authenticator.Credentials[0]
	asResp := virtualwebauthn.CreateAssertionResponse(s.rp, s.authenticator, credential, *asOpts)
	require.NoError(t, err)
	req, err = http.NewRequest(http.MethodPost, s.url+"/api/login/finish", strings.NewReader(asResp))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(nosurf.HeaderName, csrfToken)
	resp, err = s.client.Do(req)
	require.NoError(t, err)
	err = resp.Body.Close()
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	return s.GetDoc(t, "/")
}

// SubmitForm submits a form at formUrlPath with action formActionUrlPath and returns the response document.
func (s *testServer) SubmitForm(t *testing.T, formURLPath string, formActionURLPath string) *goquery.Document {
	doc := s.GetDoc(t, formURLPath)
	html, err := doc.Html()
	require.NoError(t, err)

	// Extract CSRF token from the form.
	formSelector := fmt.Sprintf("form[action='%s']", formActionURLPath)
	form := doc.Find(formSelector)
	require.Equal(t, 1, form.Length(), "form %s not found in document:\n%s", formSelector, html)
	csrfToken, ok := form.Find("input[name=csrf_token]").Attr("value")
	require.True(t, ok, "csrf_token not found in form %s", formSelector)

	// Build form data
	formData := url2.Values{}
	formData.Add("csrf_token", csrfToken)
	data := strings.NewReader(formData.Encode())

	// Submit the form
	resp, err := s.client.Post(s.url+formActionURLPath, "application/x-www-form-urlencoded", data)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		assert.NoError(t, err)
	}(resp.Body)

	// Parse the response
	doc, err = goquery.NewDocumentFromReader(resp.Body)
	require.NoError(t, err)
	return doc
}
