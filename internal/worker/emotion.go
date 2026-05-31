package worker

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/GuiCezaF/queue-processor/internal/emotion"
	"github.com/GuiCezaF/queue-processor/internal/rabbitmq"
)

type EmotionStore interface {
	CreateEmotion(
		ctx context.Context,
		userID int,
		emotion string,
		confidence float32,
		capturedAt time.Time,
	) error
}

type EmotionProcessor struct {
	queue      *rabbitmq.Client
	classifier *emotion.Classifier
	store      EmotionStore
	inputQueue string
}

func NewEmotionProcessor(
	queue *rabbitmq.Client,
	classifier *emotion.Classifier,
	store EmotionStore,
) *EmotionProcessor {
	return &EmotionProcessor{
		queue:      queue,
		classifier: classifier,
		store:      store,
		inputQueue: "emotion_requests",
	}
}

func (p *EmotionProcessor) Run() error {
	return p.queue.Consume(p.inputQueue, func(msg rabbitmq.Message) error {
		result, err := p.classifier.Predict(msg.Image)
		if err != nil {
			log.Println("inference error:", err)
			return err
		}

		userID, err := strconv.Atoi(msg.UserID)
		if err != nil {
			log.Printf("invalid user_id %q: %v", msg.UserID, err)
			return err
		}

		emotionName, confidence := highestEmotion(result)

		err = p.store.CreateEmotion(
			context.Background(),
			userID,
			emotionName,
			confidence,
			time.Now().UTC(),
		)
		if err != nil {
			log.Println("store error:", err)
			return err
		}

		return nil
	})
}

func highestEmotion(result *emotion.Emotions) (string, float32) {
	type candidate struct {
		name       string
		confidence float32
	}

	candidates := []candidate{
		{name: "angry", confidence: result.Angry},
		{name: "happy", confidence: result.Happy},
		{name: "neutral", confidence: result.Neutral},
		{name: "sad", confidence: result.Sad},
	}

	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.confidence > best.confidence {
			best = candidate
		}
	}

	return best.name, best.confidence
}
