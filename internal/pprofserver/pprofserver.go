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

// Launch a standard pprof server at addr.
func Launch(port string, logger *slog.Logger) {
	go func() {
		err := listenAndServe(fmt.Sprintf("localhost%s", port))
		logger.Error(err.Error())
		os.Exit(0)
	}()
}
