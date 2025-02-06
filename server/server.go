package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/brainsonchain/nema/nema"
)

// Server implements both public and private HTTP routers
type Server struct {
	log           *zap.Logger
	publicRouter  *chi.Mux
	privateRouter *chi.Mux
	nemaManager   *nema.Manager
}

func NewServer(log *zap.Logger, nemaManager *nema.Manager) *Server {
	setupRouter := func() *chi.Mux {
		router := chi.NewRouter()
		router.Use(middleware.Logger)
		router.Use(middleware.Recoverer)
		return router
	}

	publicRouter := setupRouter()
	privateRouter := setupRouter()

	s := &Server{
		log:           log,
		nemaManager:   nemaManager,
		publicRouter:  publicRouter,
		privateRouter: privateRouter,
	}

	// -------------------------------------------------------------------------
	// Public routes
	publicRouter.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	publicRouter.Get("/nema/state", s.nemaState)
	// publicRouter.Post("/nema/prompt", s.nemaPrompt)

	// -------------------------------------------------------------------------
	// Private routes (prefixed with /internal)
	privateRouter.Route("/internal", func(r chi.Router) {
		r.Post("/tweet", s.tweet)
		r.Post("/tweet/reply", s.tweetReply)
	})

	return s
}

// Start launches both the public and private servers
func (s *Server) Start(ctx context.Context, publicPort, privatePort string) error {
	errChan := make(chan error, 2)

	// Start public server
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%s", publicPort), s.publicRouter); err != nil {
			errChan <- fmt.Errorf("public server error: %w", err)
		}
	}()

	// Start private server
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf("fly-local-6pn:%s", privatePort), s.privateRouter); err != nil {
			errChan <- fmt.Errorf("private server error: %w", err)
		}
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
