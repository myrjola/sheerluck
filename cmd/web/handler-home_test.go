package main

import (
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func Test_application_home(t *testing.T) {
	server := startTestServer(t, io.Discard, testLookupEnv)
	doc := server.GetDoc(t, "/")
	require.Equal(t, 1, doc.Find("button:contains('Sign in')").Length())
	require.Equal(t, 1, doc.Find("button:contains('Register')").Length())
	require.Equal(t, 0, doc.Find("button:contains('Log out')").Length())

	doc = server.Register(t)
	require.Equal(t, 0, doc.Find("button:contains('Sign in')").Length())
	require.Equal(t, 0, doc.Find("button:contains('Register')").Length())
	require.Equal(t, 1, doc.Find("button:contains('Log out')").Length())

	// Log out and log back in
	doc = server.SubmitForm(t, "/", "/api/logout")
	require.Equal(t, 1, doc.Find("button:contains('Sign in')").Length())
	require.Equal(t, 1, doc.Find("button:contains('Register')").Length())
	require.Equal(t, 0, doc.Find("button:contains('Log out')").Length())

	doc = server.Login(t)
	require.Equal(t, 0, doc.Find("button:contains('Sign in')").Length())
	require.Equal(t, 0, doc.Find("button:contains('Register')").Length())
	require.Equal(t, 1, doc.Find("button:contains('Log out')").Length())
	// TODOs for nicer test setup:
	// - http client with
	//     - html assertion helper
	//     - form submission helper
	// - webauthn glue for setting up users for tests
}
