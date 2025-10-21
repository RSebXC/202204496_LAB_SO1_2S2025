FROM rust:1.82 AS builder
WORKDIR /usr/src/rust-api
COPY . .
RUN cargo build --release

FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /usr/src/rust-api/target/release/rust-api /app/rust-api
EXPOSE 8080
CMD ["/app/rust-api"]