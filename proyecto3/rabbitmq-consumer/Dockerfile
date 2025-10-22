FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o rabbitmq-consumer .

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/rabbitmq-consumer .
CMD ["./rabbitmq-consumer"]