package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/GuiCezaF/queue-processor/internal/db"
	"github.com/GuiCezaF/queue-processor/internal/emotion"
	"github.com/GuiCezaF/queue-processor/internal/processor"
	"github.com/GuiCezaF/queue-processor/internal/rabbitmq"
	"github.com/GuiCezaF/queue-processor/internal/routes"
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

	pool, err := db.Init(os.Getenv("POSTGRES_CONN"))

	if err != nil {
		log.Fatalf("Error creating connection pool: %s", err)
	}

	defer pool.Close()

	store := db.NewStore(pool)

	server := routes.Init()

	rabbitClient, err := rabbitmq.NewClient(os.Getenv("RABBITMQ_CONN"))
	if err != nil {
		log.Fatal(err)
	}
	defer rabbitClient.Close()

	classifier, err := emotion.NewClassifier()
	if err != nil {
		log.Fatal(err)
	}
	defer classifier.Close()

	emotionProcessor := processor.NewEmotionProcessor(
		rabbitClient,
		classifier,
		store,
	)

	go func() {
		if err := emotionProcessor.Run(); err != nil {
			log.Printf("emotion processor stopped: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	routes.Cancel(server)
}
