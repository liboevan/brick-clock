# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install chrony for compilation (needed for chronyc commands)
RUN apk add --no-cache chrony

# Copy go.mod and go.sum first, then download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY chrony_api_app.go ./

# Build arguments for version
ARG VERSION=0.1.0-dev
ARG BUILD_DATETIME
ENV VERSION=$VERSION
ENV BUILD_DATETIME=$BUILD_DATETIME

# Build the Go application with version and build datetime injected
RUN go build -ldflags "-X 'main.AppVersion=$VERSION' -X 'main.BuildDateTime=$BUILD_DATETIME'" -o chrony-api-app chrony_api_app.go

# Create VERSION file from build argument
RUN echo "$VERSION" > /app/VERSION

# Create build-info.json from build arguments
RUN echo "{\"version\":\"$VERSION\",\"buildDateTime\":\"$BUILD_DATETIME\",\"buildTimestamp\":$(date +%s),\"environment\":\"production\",\"service\":\"brick-clock\",\"description\":\"NTP Time Synchronization\"}" > /app/build-info.json

# Create config directory for brick-clock
RUN mkdir -p /etc/brick/clock
# Copy public key for JWT validation
COPY public.pem /etc/brick/clock/public.pem

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

# Copy the compiled Go binary and files from builder stage
COPY --from=builder /app/chrony-api-app /chrony-api-app
COPY --from=builder /app/VERSION /VERSION
COPY --from=builder /app/build-info.json /build-info.json
COPY --from=builder /etc/brick/clock/public.pem /etc/brick/clock/public.pem

# Copy scripts
COPY scripts/ /scripts/

# Expose ports
EXPOSE 123/udp
EXPOSE 17003

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"] 