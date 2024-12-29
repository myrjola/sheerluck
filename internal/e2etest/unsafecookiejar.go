package e2etest

import (
	"github.com/myrjola/sheerluck/internal/errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
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

func (u *unsafeCookieJar) SetCookies(url *url.URL, cookies []*http.Cookie) {
	for _, cookie := range cookies {
		cookie.Secure = false
	}
	u.jar.SetCookies(url, cookies)
}

func (u *unsafeCookieJar) Cookies(url *url.URL) []*http.Cookie {
	return u.jar.Cookies(url)
}
