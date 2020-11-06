/*
Copyright The Voyager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package providers

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"gomodules.xyz/x/log"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

const (
	URLPrefix         = "/.well-known/acme-challenge/"
	ACMEResponderPort = 56791
)

var defaultHTTPProvider = NewHTTPProviderServer()

func DefaultHTTPProvider() *HTTPProviderServer {
	return defaultHTTPProvider
}

// HTTPProviderServer implements ChallengeProvider for `http-01` challenge
// It may be instantiated without using the NewHTTPProviderServer function if
// you want only to use the default values.
type HTTPProviderServer struct {
	ChallengeHolders map[string]string
	mu               sync.Mutex
}

// NewHTTPProviderServer creates a new HTTPProviderServer on the selected interface and port.
// Setting iface and / or port to an empty string will make the server fall back to
// the "any" interface and port 80 respectively.
func NewHTTPProviderServer() *HTTPProviderServer {
	return &HTTPProviderServer{
		ChallengeHolders: make(map[string]string),
	}
}

// Present starts a web server and makes the token available at `HTTP01ChallengePath(token)` for web requests.
func (s *HTTPProviderServer) Present(domain, token, keyAuth string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ChallengeHolders[token+"@"+domain] = keyAuth
	return nil
}

// CleanUp closes the HTTP server and removes the token from `HTTP01ChallengePath(token)`
func (s *HTTPProviderServer) CleanUp(domain, token, keyAuth string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.ChallengeHolders, token+"@"+domain)
	return nil
}

func (s *HTTPProviderServer) NewServeMux() *http.ServeMux {
	// The handler validates the HOST header and request type.
	// For validation it then writes the token the server returned with the challenge
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimRight(r.RequestURI, "/")
		idx := strings.Index(token, URLPrefix)
		if idx >= 0 {
			token = token[idx+len(URLPrefix):]
		}

		s.mu.Lock()
		defer s.mu.Unlock()
		keyAuth, ok := s.ChallengeHolders[token+"@"+r.Host]

		if ok && r.Method == "GET" {
			w.Header().Add("Content-Type", "text/plain")
			_, err := w.Write([]byte(keyAuth))
			utilruntime.Must(err)
			log.Infof("[%s] Served key authentication", r.Host)
		} else {
			log.Infof("Received request for domain %s with method %s but the domain did not match any challenge. Please ensure your are passing the HOST header properly.", r.Host, r.Method)
			_, err := w.Write([]byte("TEST"))
			utilruntime.Must(err)
		}
	})
	return mux
}

func (s *HTTPProviderServer) Serve() {
	srv := &http.Server{
		Handler: s.NewServeMux(),
		Addr:    fmt.Sprintf(":%d", ACMEResponderPort),
	}
	// Once httpServer is shut down we don't want any lingering
	// connections, so disable KeepAlives.
	log.Infoln("Running http server provider...")
	srv.SetKeepAlivesEnabled(false)
	log.Fatal(srv.ListenAndServe())
}
