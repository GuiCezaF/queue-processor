package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

type Message struct {
	Image  string `json:"image"`
	UserID string `json:"user_id"`
}

func NewClient(connectionString string) (*Client, error) {
	if connectionString == "" {
		return nil, fmt.Errorf("RABBITMQ_CONN is required")
	}

	conn, err := amqp.Dial(connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to open rabbitmq channel: %w", err)
	}

	if err := ch.Qos(10, 0, false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("failed to configure qos: %w", err)
	}

	return &Client{
		Conn:    conn,
		Channel: ch,
	}, nil
}

func (c *Client) Close() {
	if c == nil {
		return
	}

	if c.Channel != nil {
		_ = c.Channel.Close()
	}

	if c.Conn != nil {
		_ = c.Conn.Close()
	}
}

func (c *Client) Publish(queueName string, payload interface{}) error {
	q, err := c.Channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if err := c.Channel.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		},
	); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

func (c *Client) Consume(queueName string, handler func(Message) error) error {
	q, err := c.Channel.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := c.Channel.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Println("waiting for messages:", queueName)

	for d := range msgs {
		var message Message

		if err := json.Unmarshal(d.Body, &message); err != nil {
			log.Println("invalid message:", err)
			_ = d.Nack(false, false)
			continue
		}

		if err := handler(message); err != nil {
			log.Println("handler error:", err)
			_ = d.Nack(false, false)
			continue
		}

		_ = d.Ack(false)
	}

	return nil
}
