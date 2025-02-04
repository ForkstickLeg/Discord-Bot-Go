# Stage 1: Build the Go binary
FROM golang:1.23.4 AS builder
WORKDIR /

# Copy Go dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy source code
COPY ./src/ ./
WORKDIR /src

# Build Go binary
RUN CGO_ENABLED=0 go build -o myapp .

# Stage 2: Run the built binary in a minimal image
FROM alpine:latest
WORKDIR /src

# Install SQLite (if needed)
RUN apk add --no-cache sqlite

# Copy the built binary
COPY --from=builder /src/myapp /app/myapp

# Copy the database file (if it exists)
COPY --from=builder /src/discordbot.db /src/discordbot.db

# Set environment variable defaults
ENV DB_PATH=/app/data/discordbot.db
ENV APP_ENV=docker

# Run the app
CMD ["./myapp"]
