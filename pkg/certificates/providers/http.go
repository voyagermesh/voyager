package providers

import (
	"net/http"
	"strings"
	"sync"

	"github.com/appscode/log"
)

const URLPrefix = "/.well-known/acme-challenge/"

func init() {
	defaultHTTPProvider.serve()
}

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
	once             sync.Once
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

func (s *HTTPProviderServer) serve() {
	// The handler validates the HOST header and request type.
	// For validation it then writes the token the server returned with the challenge
	var wg sync.WaitGroup
	go s.once.Do(
		func() {
			defer wg.Done()
			wg.Add(1)
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
					w.Write([]byte(keyAuth))
					log.Infof("[%s] Served key authentication", r.Host)
				} else {
					log.Infof("Received request for domain %s with method %s but the domain did not match any challenge. Please ensure your are passing the HOST header properly.", r.Host, r.Method)
					w.Write([]byte("TEST"))
				}
			})

			httpServer := &http.Server{
				Handler: mux,
				Addr:    ":56789",
			}
			// Once httpServer is shut down we don't want any lingering
			// connections, so disable KeepAlives.
			log.Infoln("Running http server provider...")
			httpServer.SetKeepAlivesEnabled(false)
			go httpServer.ListenAndServe()
		},
	)
	wg.Wait()
}
