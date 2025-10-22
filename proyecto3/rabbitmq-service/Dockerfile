FROM golang:1.24 AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o rabbitmq-service .

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/rabbitmq-service .
EXPOSE 9000
CMD ["./rabbitmq-service"]