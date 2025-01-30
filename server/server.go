package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/brainsonchain/nema/dbm"
	"github.com/brainsonchain/nema/nema"
)

// Server is the main server that implement a Chi router and handles HTTP
// requests.
type Server struct {
	log    *zap.Logger
	dbm    *dbm.Manager
	router *chi.Mux
	nema   nema.Neuro
}

func NewServer(log *zap.Logger, dbm *dbm.Manager, nema nema.Neuro) *Server {
	router := chi.NewRouter()

	s := &Server{
		log:    log,
		dbm:    dbm,
		router: router,
		nema:   nema,
	}

	// Set middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/nema", s.nemaState)

	return s
}

func (s *Server) Start(ctx context.Context, port string) error {
	return http.ListenAndServe(fmt.Sprintf(":%s", port), s.router)
}

// nemaState is a handler that returns the current state of the nema.
func (s *Server) nemaState(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(s.nema); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
}
