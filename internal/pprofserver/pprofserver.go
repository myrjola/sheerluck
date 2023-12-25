package pprofserver

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"os"
)

func Handle(mux *http.ServeMux) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
}

func newServeMux() *http.ServeMux {
	mux := http.NewServeMux()
	Handle(mux)
	return mux
}

func newServer(addr string) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: newServeMux(),
	}
}

func listenAndServe(addr string) error {
	return newServer(addr).ListenAndServe()
}

// Launch a standard pprof server at ipv6 loopback address ::1 and given port.
func Launch(port string, logger *slog.Logger) {
	go func() {
		addr := fmt.Sprintf("[::1]%s", port)
		logger.Info("starting pprof server", "addr", addr)
		err := listenAndServe(addr)
		logger.Error(err.Error())
		os.Exit(0)
	}()
}
