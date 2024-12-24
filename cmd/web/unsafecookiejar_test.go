package main

import (
	"github.com/myrjola/sheerluck/internal/errors"
	"net/http"
	"net/http/cookiejar"
	url2 "net/url"
)

type unsafeCookieJar struct {
	jar *cookiejar.Jar
}

// newUnsafeCookieJar returns a [http.CookieJar] that does not enforce the Secure flag this is useful for testing.
func newUnsafeCookieJar() (*unsafeCookieJar, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, errors.Wrap(err, "new cookie jar")
	}

	return &unsafeCookieJar{jar: jar}, nil
}

func (u *unsafeCookieJar) SetCookies(url *url2.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		cookie.Secure = false
	}
	u.jar.SetCookies(url, cookies)
}

func (u *unsafeCookieJar) Cookies(url *url2.URL) []*http.Cookie {
	return u.jar.Cookies(url)
}
