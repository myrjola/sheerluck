package main

import (
	"net/http"
	"time"
)

const timeoutBody = `<html lang="en">
<head><title>Timeout</title></head>
<body>
<h1>Timeout</h1>
<div>
    <button type="button">
        <span>Retry</span>
        <script>
          document.currentScript.parentElement.addEventListener('click', function () {
            location.reload();
          });
        </script>
    </button>
</div>
</body>
</html>
`

// timeoutHandler responds with a 503 Service Unavailable error when the handler does not meet the deadline.
func timeoutHandler(h http.Handler, defaultTimeout time.Duration) http.Handler {
	// We want the timeout to be a little shorter than the server's read timeout so that the
	// timeout handler has a chance to respond before the server closes the connection.
	httpHandlerTimeout := defaultTimeout - 500*time.Millisecond //nolint:mnd // 500ms
	return http.TimeoutHandler(h, httpHandlerTimeout, timeoutBody)
}
