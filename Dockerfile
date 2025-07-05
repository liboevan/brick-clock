# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install chrony for compilation (needed for chronyc commands)
RUN apk add --no-cache chrony

# Copy Go source code and version file
COPY chrony_api_app.go .
COPY VERSION .

# Build arguments for version
ARG VERSION=0.1.0-dev
ARG BUILD_DATETIME
ENV VERSION=$VERSION
ENV BUILD_DATETIME=$BUILD_DATETIME

# Build the Go application with version and build datetime injected
RUN go mod init el/brick-clock && \
    go build -ldflags "-X 'main.AppVersion=$VERSION' -X 'main.BuildDateTime=$BUILD_DATETIME'" -o chrony-api-app chrony_api_app.go

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

# Copy scripts
COPY scripts/ /scripts/

# Create VERSION file from build argument
RUN echo "$VERSION" > /VERSION

# Expose ports
EXPOSE 123/udp
EXPOSE 17003

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"] 