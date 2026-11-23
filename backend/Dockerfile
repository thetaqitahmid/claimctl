# Build Stage
FROM golang:1.25 AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o main cmd/main.go

# Run Stage
FROM alpine:latest

# Create non-root user and group
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Install necessary runtime dependencies (if any)
RUN apk --no-cache add ca-certificates

# Copy the binary from the builder stage
COPY --from=builder /app/main .

# Copy database migrations and seed if needed at runtime
COPY --from=builder /app/migrations ./migrations

# Change ownership of the app directory to non-root user
RUN chown -R appuser:appgroup /app

# Use writable locations when the container runs with a read-only root
# filesystem. Some dependencies create cache directories at startup.
ENV HOME=/tmp \
    XDG_CACHE_HOME=/tmp/.cache

# Switch to non-root user
USER appuser

EXPOSE 3000

CMD ["./main"]
