package httpserver

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(addr string) *http.Server {
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.RequestID)
	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(middleware.Recoverer)

	registerRoutes(router)

	return &http.Server{
		Addr:           addr,
		Handler:        router,
		MaxHeaderBytes: 1 << 20,
	}
}

func Shutdown(server *http.Server) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
}
