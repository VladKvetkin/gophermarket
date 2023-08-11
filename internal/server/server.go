package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/VladKvetkin/gophermart/internal/config"
	"github.com/VladKvetkin/gophermart/internal/handler"
	"github.com/VladKvetkin/gophermart/internal/storage"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

type Server struct {
	config  config.Config
	mux     chi.Router
	server  *http.Server
	storage storage.Storage
}

func NewServer(config config.Config, storage storage.Storage) *Server {
	mux := chi.NewMux()

	return &Server{
		config:  config,
		mux:     mux,
		storage: storage,
		server: &http.Server{
			Addr:              config.Address,
			Handler:           mux,
			ReadTimeout:       5 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      5 * time.Second,
			IdleTimeout:       5 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	s.setupRoutes(handler.NewHandler(s.storage))

	zap.L().Info("starting server", zap.String("address", s.config.Address))

	if err := s.server.ListenAndServe(); err != nil {
		return fmt.Errorf("error starting server: %w", err)
	}

	return nil
}

func (s *Server) Stop() error {
	zap.L().Info("stopping server")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("error stopping server: %w", err)
	}

	return nil
}
