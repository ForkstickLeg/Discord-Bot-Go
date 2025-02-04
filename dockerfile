# Stage 1: Build the Go binary
FROM golang:1.21 AS builder
WORKDIR /app

# Copy Go dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy source code
COPY src/ ./src/
WORKDIR /app/src

# Build Go binary
RUN CGO_ENABLED=0 go build -o myapp main.go

# Stage 2: Run the built binary in a minimal image
FROM alpine:latest
WORKDIR /app

# Install SQLite (if needed)
RUN apk add --no-cache sqlite

# Copy the built binary
COPY --from=builder /app/src/myapp /app/myapp

# Ensure the data directory exists
RUN mkdir -p /app/data

# Copy the database file (if it exists)
COPY --from=builder /app/src/discordbot.db /app/data/discordbot.db

# Set environment variable defaults
ENV DB_PATH=/app/data/discordbot.db
ENV APP_ENV=docker

# Run the app
CMD ["./myapp"]
