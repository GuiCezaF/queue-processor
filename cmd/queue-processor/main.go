package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/GuiCezaF/queue-processor/internal/config"
	"github.com/GuiCezaF/queue-processor/internal/emotion"
	"github.com/GuiCezaF/queue-processor/internal/httpserver"
	"github.com/GuiCezaF/queue-processor/internal/rabbitmq"
	"github.com/GuiCezaF/queue-processor/internal/storage/postgres"
	"github.com/GuiCezaF/queue-processor/internal/worker"
	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("error loading .env file: %v", err)
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	pool, err := postgres.Open(cfg.PostgresConn)
	if err != nil {
		log.Fatalf("error creating PostgreSQL pool: %s", err)
	}
	defer pool.Close()

	store := postgres.NewStore(pool)
	server := httpserver.New(cfg.HTTPAddr)

	rabbitClient, err := rabbitmq.NewClient(cfg.RabbitMQConn)
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitClient.Close()

	classifier, err := emotion.NewClassifier()
	if err != nil {
		log.Fatal(err)
	}
	defer classifier.Close()

	emotionProcessor := worker.NewEmotionProcessor(
		rabbitClient,
		classifier,
		store,
	)

	go func() {
		log.Printf("HTTP server running on %s", cfg.HTTPAddr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	go func() {
		if err := emotionProcessor.Run(); err != nil {
			log.Printf("emotion processor stopped: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	httpserver.Shutdown(server)
}
