# Multi-stage build for smaller final image
FROM golang:alpine AS builder

# Install build dependencies for SQLite
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
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o towing-server .

# Final stage - minimal image
FROM alpine:latest

# Install sqlite (required for runtime)
RUN apk --no-cache add ca-certificates sqlite

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/towing-server .

# Expose port 8080
EXPOSE 8080

# Run the application
CMD ["./towing-server"]