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

# Run the app
CMD echo "APP_ID: $APP_ID, BOT_TOKEN: $BOT_TOKEN" && ./myapp
