package processor

import (
	"log"

	"github.com/GuiCezaF/queue-processor/internal/emotion"
	"github.com/GuiCezaF/queue-processor/internal/rabbitmq"
)

type EmotionResponse struct {
	UserID  string            `json:"user_id"`
	Emotion *emotion.Emotions `json:"emotion"`
}

type EmotionProcessor struct {
	queue       *rabbitmq.Client
	classifier  *emotion.Classifier
	inputQueue  string
	outputQueue string
}

func NewEmotionProcessor(
	queue *rabbitmq.Client,
	classifier *emotion.Classifier,
) *EmotionProcessor {
	return &EmotionProcessor{
		queue:       queue,
		classifier:  classifier,
		inputQueue:  "emotion_requests",
		outputQueue: "emotion_results",
	}
}

func (p *EmotionProcessor) Run() error {
	return p.queue.Consume(p.inputQueue, func(msg rabbitmq.Message) error {
		result, err := p.classifier.Predict(msg.Image)
		if err != nil {
			log.Println("inference error:", err)
			return err
		}

		response := EmotionResponse{
			UserID:  msg.UserID,
			Emotion: result,
		}

		if err := p.queue.Publish(p.outputQueue, response); err != nil {
			log.Println("publish error:", err)
			return err
		}

		log.Printf("emotion processed for user %s : %+v", msg.UserID, response.Emotion)
		return nil
	})
}
