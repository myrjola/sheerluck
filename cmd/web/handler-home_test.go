package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/descope/virtualwebauthn"
	"github.com/justinas/nosurf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	url2 "net/url"
	"strings"
	"testing"
)

func Test_application_home(t *testing.T) {
	url := startTestServer(t, io.Discard, testLookupEnv)
	jar, err := newUnsafeCookieJar()
	require.NoError(t, err)
	client := &http.Client{Jar: jar}

	res, err := client.Get(url)
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

	// Find the form
	registrationStartURLPath := "/api/registration/start"
	formSelector := fmt.Sprintf("form[action='%s']", registrationStartURLPath)
	form := doc.Find(formSelector)
	// Find the CSRF token
	csrfToken, ok := form.Find("input[name=csrf_token]").Attr("value")
	require.True(t, ok, "csrf_token not found in form %s", formSelector)

	req, err := http.NewRequest(http.MethodPost, url+registrationStartURLPath, nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(nosurf.HeaderName, csrfToken)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err := io.ReadAll(resp.Body)
	err = resp.Body.Close()
	require.NoError(t, err)
	attOpts, err := virtualwebauthn.ParseAttestationOptions(string(bodyBytes))
	require.NoError(t, err)
	attestationResponse := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attOpts)
	req, err = http.NewRequest(http.MethodPost, url+"/api/registration/finish", strings.NewReader(attestationResponse))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(nosurf.HeaderName, csrfToken)
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// At this point, our credential is ready for logging in.
	authenticator.AddCredential(credential)
	// This option is needed for making Passkey login work.
	authenticator.Options.UserHandle = []byte(attOpts.UserID)

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

	// Log out and log back in
	doc = submitForm(t, client, url, "/api/logout", doc)

	// Find the form
	loginStartURLPath := "/api/login/start"
	formSelector = fmt.Sprintf("form[action='%s']", loginStartURLPath)
	form = doc.Find(formSelector)
	// Find the CSRF token
	csrfToken, ok = form.Find("input[name=csrf_token]").Attr("value")
	require.True(t, ok, "csrf_token not found in form %s", formSelector)

	req, err = http.NewRequest(http.MethodPost, url+loginStartURLPath, nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(nosurf.HeaderName, csrfToken)
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	bodyBytes, err = io.ReadAll(resp.Body)
	err = resp.Body.Close()
	require.NoError(t, err)
	asOpts, err := virtualwebauthn.ParseAssertionOptions(string(bodyBytes))

	require.NoError(t, err)
	asResp := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *asOpts)
	require.NoError(t, err)
	req, err = http.NewRequest(http.MethodPost, url+"/api/login/finish", strings.NewReader(asResp))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(nosurf.HeaderName, csrfToken)
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	// TODOs for nicer test setup:
	// - http client with
	//     - html assertion helper
	//     - form submission helper
	// - webauthn glue for setting up users for tests
}

func submitForm(t *testing.T, client *http.Client, baseURL, url string, doc *goquery.Document) *goquery.Document {
	html, err := doc.Html()
	require.NoError(t, err)

	// Find the form
	formSelector := fmt.Sprintf("form[action='%s']", url)
	form := doc.Find(formSelector)
	require.Equal(t, 1, form.Length(), "form %s not found in document:\n%s", formSelector, html)

	// Find the CSRF token
	csrfToken, ok := form.Find("input[name=csrf_token]").Attr("value")
	require.True(t, ok, "csrf_token not found in form %s", formSelector)

	// Build form data
	formData := url2.Values{}
	formData.Add("csrf_token", csrfToken)
	data := strings.NewReader(formData.Encode())

	// Submit the form
	resp, err := client.Post(baseURL+url, "application/x-www-form-urlencoded", data)
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
