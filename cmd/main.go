package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GuiCezaF/queue-processor/internal/queue"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

type sendResponse struct {
	Msg string `json:"msg"`
}

func init() {

	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {

	queue.NewRabbitMQConnection(os.Getenv("RABBITMQ_CONN"))

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(middleware.RequestID)
	r.Use(middleware.ClientIPFromRemoteAddr)
	r.Use(middleware.Recoverer)

	s := &http.Server{
		Addr:           ":8080",
		Handler:        r,
		MaxHeaderBytes: 11 << 20,
	}

	r.Get("/ping", ping)

	go func() {
		log.Println("HTTP server running on :8080")

		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	go func() {
		err := queue.RabbitMQClient.ConsumeQueue("my_queue", func(msg queue.Message) error {

			fmt.Println("Received message:", msg.Data)

			// TODO: Process message

			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}

}

func ping(w http.ResponseWriter, req *http.Request) {

	r := sendResponse{Msg: "Pong"}
	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
