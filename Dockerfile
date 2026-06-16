# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy source code (includes go.mod)
COPY . .
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o labelarr ./cmd/labelarr

# Runtime stage
FROM alpine:3.22

# Install ca-certificates for HTTPS requests and debugging tools
# Using retry logic for QEMU emulation stability during multi-arch builds
RUN apk add --no-cache --update ca-certificates tzdata bash curl wget && \
    ln -sf /bin/bash /bin/sh

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/labelarr .

# Create a non-root user
RUN adduser -D -s /bin/bash labelarr
USER labelarr

# Webhook server port (only used when WEBHOOK_ENABLED=true)
EXPOSE 9090

CMD ["./labelarr"] 
