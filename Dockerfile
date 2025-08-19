# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o templepoints .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates sqlite

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/templepoints .

# Copy static files
COPY --from=builder /app/*.html ./

# Create directory for database
RUN mkdir -p /app/data

# Expose port
EXPOSE 8080

# Set environment variable for database location
ENV DATABASE_PATH=/app/data/templepoints.db

# Run the application
CMD ["./templepoints"]