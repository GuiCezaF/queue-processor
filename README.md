# Queue Processor

Queue Processor is a Go-based background service that consumes image-processing jobs from RabbitMQ, runs emotion classification with an ONNX model, and stores the results in PostgreSQL.

The service also exposes a small HTTP endpoint for health checks.

## Overview

On startup, the application:

1. Loads configuration from environment variables.
2. Connects to PostgreSQL and creates the required tables if they do not already exist.
3. Connects to RabbitMQ.
4. Loads the ONNX emotion classification model.
5. Starts consuming messages from the `emotion_requests` queue.

For each message, the worker:

1. Decodes the input image from Base64.
2. Runs emotion inference.
3. Selects the most confident emotion.
4. Persists the result in PostgreSQL.

## Features

- RabbitMQ consumer with manual acknowledgements.
- ONNX Runtime-based emotion classifier.
- PostgreSQL persistence for emotion logs.
- Graceful shutdown on `SIGINT` and `SIGTERM`.
- HTTP health endpoint at `/ping`.

## Requirements

- Go 1.26.3 or newer
- PostgreSQL
- RabbitMQ
- ONNX Runtime shared library
- An ONNX emotion model file

## Configuration

The application reads the following environment variables:

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `HTTP_ADDR` | No | `:8080` | Address used by the HTTP server. |
| `RABBITMQ_CONN` | Yes | - | RabbitMQ connection string. |
| `POSTGRES_CONN` | Yes | - | PostgreSQL connection string. |
| `MODEL_PATH` | No | `./assets/models/emotion_model.onnx` | Path to the ONNX model file. |
| `ONNXRUNTIME_SHARED_LIBRARY_PATH` | No | Auto-detected | Full path to the ONNX Runtime shared library. |

### Example `.env`

```env
HTTP_ADDR=:8080
RABBITMQ_CONN=amqp://guest:guest@localhost:5672/
POSTGRES_CONN=postgres://postgres:postgres@localhost:5432/emotion_db?sslmode=disable
MODEL_PATH=./assets/models/emotion_model.onnx
ONNXRUNTIME_SHARED_LIBRARY_PATH=/usr/local/lib/libonnxruntime.so.1.25.0
```

## Message Format

The worker consumes JSON messages from the `emotion_requests` queue with the following schema:

```json
{
  "image": "base64-encoded-image",
  "user_id": "123"
}
```

Notes:

- The `image` field may be a raw Base64 string or a data URL.
- The `user_id` field is parsed as an integer before persistence.

## Database Schema

The application creates the following tables automatically if they do not exist:

- `users`
- `emotion_logs`

The `emotion_logs` table stores:

- `user_id`
- `emotion`
- `confidence`
- `captured_at`

## Running with Docker

The repository includes a `docker-compose.yaml` file that starts:

- the worker service
- PostgreSQL
- RabbitMQ with the management UI

### Steps

1. Place the ONNX model file at `assets/models/emotion_model.onnx`.
2. Create a `.env` file with the required connection variables.
3. Start the stack:

```bash
docker compose up --build
```

### Default ports

- Worker HTTP server: `8080`
- PostgreSQL: `5433` mapped to container port `5432`
- RabbitMQ AMQP: `5672`
- RabbitMQ Management UI: `15672`

## Running Locally

If you prefer to run the service without Docker, make sure the required dependencies are available on your machine:

- PostgreSQL must be reachable via `POSTGRES_CONN`
- RabbitMQ must be reachable via `RABBITMQ_CONN`
- The ONNX model must exist at `MODEL_PATH`
- The ONNX Runtime shared library must be installed and discoverable

Then run:

```bash
go run ./cmd/queue-processor
```

## Health Check

The HTTP server exposes:

- `GET /ping`

Example response:

```json
{
  "msg": "Pong"
}
```

## Project Structure

- `cmd/queue-processor` - application entrypoint
- `internal/config` - environment-based configuration loading
- `internal/httpserver` - HTTP server and routes
- `internal/rabbitmq` - RabbitMQ client and queue consumption
- `internal/emotion` - ONNX Runtime model loading and inference
- `internal/storage/postgres` - PostgreSQL connection, migrations, and persistence
- `internal/worker` - queue message processing pipeline
- `assets/models` - model placement instructions and runtime model files

## Notes

- The worker uses the `emotion_requests` queue by default.
- If a message fails validation, inference, or persistence, it is negatively acknowledged by RabbitMQ and not requeued.
- The ONNX Runtime shared library can be provided explicitly with `ONNXRUNTIME_SHARED_LIBRARY_PATH` or discovered automatically from common locations.

