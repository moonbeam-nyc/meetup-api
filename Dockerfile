# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-w -s" -o /app/meetup-api ./cmd/server

# Production stage - use scratch for minimal size
FROM scratch

# Copy CA certificates for HTTPS
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/meetup-api /meetup-api

# Expose port
EXPOSE 3000

# Run as non-root (nobody user)
USER 65534:65534

# Start application
ENTRYPOINT ["/meetup-api"]
