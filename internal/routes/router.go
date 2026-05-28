package routes

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func Init() *http.Server {
	r, s := configureRoutesAndServer()
	registerRoutes(r)
	go func() {
		log.Println("HTTP server running on :8080")

		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	return s
}

func Cancel(server *http.Server) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
}

func configureRoutesAndServer() (*chi.Mux, *http.Server) {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.RequestID)
	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(middleware.Recoverer)

	server := &http.Server{
		Addr:           ":8080",
		Handler:        router,
		MaxHeaderBytes: 1 << 20,
	}

	return router, server
}

func registerRoutes(r *chi.Mux) {
	r.Get("/ping", ping)
}
