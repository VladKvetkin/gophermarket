package server

import (
	"compress/gzip"
	"net/http"

	"github.com/VladKvetkin/gophermart/internal/handler"
	"github.com/VladKvetkin/gophermart/internal/middleware"
	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func (s *Server) setupRoutes(handler *handler.Handler) {
	s.setupMiddleware()

	s.mux.Route("/", func(r chi.Router) {
		r.Route("/api", func(r chi.Router) {
			r.Route("/user", func(r chi.Router) {
				r.Post("/login", http.HandlerFunc(handler.Login))
				r.Post("/register", http.HandlerFunc(handler.Register))

				r.Post("/orders", http.HandlerFunc(handler.SaveOrder))
				r.Get("/orders", http.HandlerFunc(handler.GetOrders))

				r.Route("/balance", func(r chi.Router) {
					r.Get("/", http.HandlerFunc(handler.GetBalance))
					r.Post("/withdraw", http.HandlerFunc(handler.Withdraw))
				})

				r.Get("/withdrawals", http.HandlerFunc(handler.GetWithdrawals))
			})
		})
	})
}

func (s *Server) setupMiddleware() {
	s.mux.Use(
		middleware.DecompressBodyReader,
		middleware.Auth,
		middleware.Logger,
		chiMiddleware.Compress(gzip.BestCompression, "application/json", "text/html", "text/plain"),
	)
}
