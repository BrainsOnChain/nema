package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/brainsonchain/nema/nema"
)

// Server is the main server that implement a Chi router and handles HTTP
// requests.
type Server struct {
	log         *zap.Logger
	router      *chi.Mux
	nemaManager *nema.Manager
}

func NewServer(log *zap.Logger, nemaManager *nema.Manager) *Server {
	router := chi.NewRouter()

	s := &Server{
		log:         log,
		nemaManager: nemaManager,
		router:      router,
	}

	// Set middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/nema/state", s.nemaState)
	router.Post("/nema/prompt", s.nemaPrompt)

	return s
}

func (s *Server) Start(ctx context.Context, port string) error {
	return http.ListenAndServe(fmt.Sprintf(":%s", port), s.router)
}

// nemaState is a handler that returns the current state of the nema.
func (s *Server) nemaState(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(s.nemaManager.GetState()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}

// nemaPrompt is a handler that takes a incoming prompt, asks the LLM, and
// returns the response.
func (s *Server) nemaPrompt(w http.ResponseWriter, r *http.Request) {
	// Read the request body
	var prompt struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&prompt); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.log.Info("incoming prompt", zap.String("prompt", prompt.Prompt))

	response, err := s.nemaManager.AskLLM(r.Context(), prompt.Prompt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type resp struct {
		HumanMessage string `json:"human_message"`
	}

	jsonResp := resp{
		HumanMessage: response.HumanMessage,
	}

	if err := json.NewEncoder(w).Encode(jsonResp); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
