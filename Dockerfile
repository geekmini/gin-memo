# Build stage
FROM golang:alpine AS builder

# Install git (needed for go mod download)
RUN apk add --no-cache git

WORKDIR /app

# Copy go.mod and go.sum first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
# CGO_ENABLED=0 for static binary (no C dependencies)
# -ldflags="-s -w" strips debug info for smaller binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server

# Runtime stage
FROM alpine:3.19

# Add ca-certificates for HTTPS and tzdata for timezones
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Create non-root user for security
RUN adduser -D -g '' appuser
USER appuser

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./server"]
