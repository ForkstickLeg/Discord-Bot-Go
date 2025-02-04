# Stage 1: Build the Go binary
FROM golang:1.23.4 AS builder
WORKDIR /app

# Set Go environment variables
ENV GOPATH=/go
ENV GO111MODULE=on

# Copy Go dependencies
COPY go.mod go.sum ./
RUN go mod tidy

# Copy source code correctly (since src/ is removed)
COPY . ./

# Build Go binary
RUN CGO_ENABLED=0 go build -o myapp .

# Stage 2: Run the built binary in a minimal image
FROM alpine:latest
WORKDIR /app

# Install SQLite (if needed)
RUN apk add --no-cache sqlite

# Copy the built binary
COPY --from=builder /app/myapp /app/myapp

# Set environment variable defaults
ENV DB_PATH=./discordbot.db
ENV APP_ENV=production

# Run the app
CMD ["./myapp"]
