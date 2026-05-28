package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GuiCezaF/queue-processor/internal/emotion"
	"github.com/GuiCezaF/queue-processor/internal/processor"
	"github.com/GuiCezaF/queue-processor/internal/rabbitmq"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

type sendResponse struct {
	Msg string `json:"msg"`
}

type testEmotionRequest struct {
	Image  string `json:"image"`
	UserID string `json:"user_id"`
}

func init() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("error loading .env file: %v", err)
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Use(middleware.Timeout(60 * time.Second))
	router.Use(middleware.RequestID)
	router.Use(middleware.ClientIPFromRemoteAddr)
	router.Use(middleware.Recoverer)

	rabbitClient, err := rabbitmq.NewClient(os.Getenv("RABBITMQ_CONN"))
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitClient.Close()

	classifier, err := emotion.NewClassifier(modelPath())
	if err != nil {
		log.Fatal(err)
	}
	defer classifier.Close()

	emotionProcessor := processor.NewEmotionProcessor(
		rabbitClient,
		classifier,
	)

	go func() {
		if err := emotionProcessor.Run(); err != nil {
			log.Printf("emotion processor stopped: %v", err)
		}
	}()

	router.Get("/ping", ping)
	router.Post("/debug/emotions", func(w http.ResponseWriter, req *http.Request) {
		var payload testEmotionRequest

		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if payload.Image == "" || payload.UserID == "" {
			http.Error(w, "image and user_id are required", http.StatusBadRequest)
			return
		}

		if err := rabbitClient.Publish("emotion_requests", rabbitmq.Message{
			Image:  payload.Image,
			UserID: payload.UserID,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)

		if err := json.NewEncoder(w).Encode(sendResponse{Msg: "message enqueued"}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	server := &http.Server{
		Addr:           ":8080",
		Handler:        router,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Println("HTTP server running on :8080")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
}

func ping(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(sendResponse{Msg: "Pong"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func modelPath() string {
	if path := os.Getenv("MODEL_PATH"); path != "" {
		return path
	}

	return "./assets/models/emotion_model.onnx"
}
