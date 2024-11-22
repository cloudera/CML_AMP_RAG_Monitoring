package sbhttpserver

import (
	"github.com/dimfeld/httptreemux"
	"net/http"
)

func envoyPreRouter(w http.ResponseWriter, r *http.Request, config *Config, router *httptreemux.TreeMux) {
	if config.KeepEnvoyPath {
		if r.Header.Get("x-envoy-original-path") != "" {
			r.URL.Path = r.Header.Get("x-envoy-original-path")
			r.RequestURI = r.Header.Get("x-envoy-original-path")
		}
	}
	router.ServeHTTP(w, r)
}
