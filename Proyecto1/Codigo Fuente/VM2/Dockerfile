FROM golang:1.22 as builder
WORKDIR /app
COPY . .
RUN go build -o api3 main.go

FROM scratch
COPY --from=builder /app/api3 /
EXPOSE 8083
CMD ["/api3"]