FROM golang:1.26.3-bookworm AS builder

ARG ONNXRUNTIME_VERSION=1.25.0

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    gcc \
    g++ \
    libc6-dev \
    tar \
    wget \
  && rm -rf /var/lib/apt/lists/*

RUN wget -q "https://github.com/microsoft/onnxruntime/releases/download/v${ONNXRUNTIME_VERSION}/onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}.tgz" \
  && tar -xzf "onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}.tgz" \
  && mv "onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}" /opt/onnxruntime \
  && rm "onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}.tgz"

COPY go.mod go.sum ./

RUN go mod download

COPY . .

ENV CGO_ENABLED=1 \
    CGO_CFLAGS="-I/opt/onnxruntime/include" \
    CGO_LDFLAGS="-L/opt/onnxruntime/lib -lonnxruntime"

RUN go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/queue-processor

FROM debian:bookworm-slim AS runtime

ARG ONNXRUNTIME_VERSION=1.25.0

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libgomp1 \
    libstdc++6 \
  && rm -rf /var/lib/apt/lists/* \
  && useradd --system --uid 10001 --home-dir /nonexistent --shell /usr/sbin/nologin appuser

COPY --from=builder /out/worker /usr/local/bin/worker
COPY --from=builder /opt/onnxruntime/lib/libonnxruntime.so* /usr/local/lib/

RUN ldconfig

ENV MODEL_PATH=/models/emotion_model.onnx \
    ONNXRUNTIME_SHARED_LIBRARY_PATH=/usr/local/lib/libonnxruntime.so.${ONNXRUNTIME_VERSION} \
    LD_LIBRARY_PATH=/usr/local/lib

USER appuser

ENTRYPOINT ["/usr/local/bin/worker"]
