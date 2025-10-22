FROM rust:1.82 AS builder
WORKDIR /usr/src/rust-api
COPY . .
RUN cargo build --release

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y libssl3 ca-certificates && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /usr/src/rust-api/target/release/rust-api /app/rust-api
EXPOSE 8080
CMD ["/app/rust-api"]