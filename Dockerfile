# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install chrony for compilation (needed for chronyc commands)
RUN apk add --no-cache chrony

# Copy Go source code
COPY chrony_api_app.go .

# Build the Go application
RUN go mod init el/chrony-suite && \
    go build -o chrony-api-app chrony_api_app.go

# Runtime stage
FROM alpine:latest

# Install chrony and other dependencies
RUN apk update && \
    apk add --no-cache chrony

# Copy chrony configuration
COPY chrony.conf /etc/chrony/chrony.conf

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Copy the compiled Go binary from builder stage
COPY --from=builder /app/chrony-api-app /chrony-api-app

# Expose ports
EXPOSE 123/udp
EXPOSE 8291

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"] 