package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/brainsonchain/nema/dbm"
)

// Server is the main server that implement a Chi router and handles HTTP
// requests.
type Server struct {
	log    *zap.Logger
	dbm    *dbm.Manager
	router *chi.Mux
}

func NewServer(log *zap.Logger, dbm *dbm.Manager) *Server {
	router := chi.NewRouter()

	// Set middleware
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})

	return &Server{
		log:    log,
		dbm:    dbm,
		router: router,
	}
}

func (s *Server) Start(ctx context.Context, port string) error {
	return http.ListenAndServe(fmt.Sprintf(":%s", port), s.router)
}
