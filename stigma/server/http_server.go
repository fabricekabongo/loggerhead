package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/fabricekabongo/loggerhead/stigma"
)

// HTTPServer provides minimal wrappers around the stigma engine.
type HTTPServer struct {
	engine *stigma.Engine
}

// NewHTTPServer constructs the server.
func NewHTTPServer(engine *stigma.Engine) *HTTPServer {
	return &HTTPServer{engine: engine}
}

// ServeHTTP dispatches requests.
func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/ingest":
		s.handleIngest(w, r)
	case "/query/stigma-map":
		s.handleQuery(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *HTTPServer) handleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var samples []stigma.LocationSample
	if err := json.NewDecoder(r.Body).Decode(&samples); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	if err := s.engine.IngestBatch(samples); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *HTTPServer) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var q stigma.StigmaMapQuery
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	res, err := s.engine.QueryStigmaMap(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Println("failed to encode response", err)
	}
}
