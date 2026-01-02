FROM golang:1.26rc1 AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=1 go build -o /app/astera ./cmd/astera.go

EXPOSE 8080
ENTRYPOINT ["/app/astera"]
