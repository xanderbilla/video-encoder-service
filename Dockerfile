FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server cmd/api/main.go

# Final stage
FROM alpine:latest

# Install FFmpeg and runtime dependencies
RUN apk --no-cache add ffmpeg ca-certificates wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Create necessary directories
RUN mkdir -p videos outputs chunks tmp cache logs

# Expose port
EXPOSE 8080

# Run the server
CMD ["./server"]
