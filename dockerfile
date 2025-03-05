# Stage 1: Build the Go binary
FROM golang:1.23.4 AS builder
WORKDIR /app

# Set Go environment variables
ENV GOPATH=/go
ENV GO111MODULE=on

# Copy Go dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy source code
COPY . ./

# Build Go binary
RUN CGO_ENABLED=0 go build -o myapp .

# Stage 2: Run the built binary in a minimal image
FROM debian:bookworm-slim
WORKDIR /app

# Install SQLite (if needed)
RUN apt update && apt install -y sqlite3

RUN 

# Copy the built binary
COPY --from=builder /app/myapp /app/myapp

# Create env file
RUN echo '#!/bin/sh' > /app/start.sh && \
    echo 'echo "APP_ID=$APP_ID" > /app/.env' >> /app/start.sh && \
    echo 'echo "APP_ENV=$APP_ENV" > /app/.env' >> /app/start.sh && \
    echo 'echo "BOT_TOKEN=$BOT_TOKEN" >> /app/.env' >> /app/start.sh && \
    echo './myapp' >> /app/start.sh && \
    chmod +x /app/start.sh

# Run the app
CMD ["./myapp"]