# Dockerfile
FROM golang:1.20-alpine AS build

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /infopulse-node ./cmd/main.go

# Use a small image for the final container
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy the binary from the build stage
COPY --from=build /infopulse-node .

# Copy config directory
COPY --from=build /app/config ./config

# Create data directory
RUN mkdir -p /data /var/log

# Set environment variables
ENV TZ=UTC

# Set volume for persistent data
VOLUME ["/app/config", "/data", "/var/log"]

# Run the application
CMD ["./infopulse-node", "--config", "./config/config.json"]
