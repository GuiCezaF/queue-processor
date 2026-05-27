package queue

import (
	"encoding/json"
	"fmt"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
}

type Message struct {
	Data string
}

var RabbitMQClient *RabbitMQ

func NewRabbitMQConnection(connection_string string) {
	conn, err := amqp.Dial(connection_string)
	if err != nil {
		log.Fatalf("Failed to connet to rabbitmq: %s", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a rabbitmq channel: %s", err)
	}

	RabbitMQClient = &RabbitMQ{
		Conn:    conn,
		Channel: ch,
	}
}

func (r *RabbitMQ) SendToQueue(message Message, rabbitmq_queue string) error {

	q, err := r.Channel.QueueDeclare(
		rabbitmq_queue, // queue name
		true,           // durable
		false,          // delete when unused
		false,          // exclusive
		false,          // no-wait
		nil,            // arguments
	)

	failOnError(err, "Failed to declare RabbitMQ queue")

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	err = r.Channel.Publish(
		"",     // exchange
		q.Name, // routing key (queue name)
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		return fmt.Errorf("failed to publish message: %v", err)
	}

	log.Printf("Message has been sent to RabbitMQ queue: %s", message)

	return nil
}

func (r *RabbitMQ) ConsumeQueue(
	rabbitmq_queue string,
	handler func(Message) error,
) error {
	q, err := r.Channel.QueueDeclare(
		rabbitmq_queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)

	failOnError(err, "Failed to declare RabbitMQ queue")

	msgs, err := r.Channel.Consume(
		q.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return fmt.Errorf("Failed to Consume Queue: %s", err)
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			var message Message

			err := json.Unmarshal(d.Body, &message)
			if err != nil {
				d.Nack(false, false)
				continue
			}

			err = handler(message)
			if err != nil {
				d.Nack(false, false)
				continue
			}

			d.Ack(false)
		}
	}()

	log.Println("Waiting for messages...")

	<-forever
	return nil
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}
