# Build Stage
FROM golang:alpine AS builder

# Install build dependencies (needed for SQLite CGO)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=1 is required for go-sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o server cmd/server/main.go

# Final Stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
# sqlite-libs might be needed depending on how it's linked, but usually static ifmusl-dev used correctly.
# Adding ca-certificates for potential HTTPS requests.
RUN apk add --no-cache ca-certificates sqlite-libs curl tzdata

# Set Timezone
ENV TZ=Asia/Dhaka

# Copy binary from builder
COPY --from=builder /app/server .

# Copy web assets (templates & static)
COPY --from=builder /app/web ./web

# Create storage directory
RUN mkdir -p storage

# Expose port
EXPOSE 8080

# Create a non-root user
# RUN adduser -D -g '' appuser

# Set ownership of the storage directory
# RUN chown -R appuser:appuser /app/storage

# Switch to the non-root user
# USER appuser

# Run the binary
CMD ["./server"]
