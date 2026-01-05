# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy source code (includes go.mod)
COPY . .
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o labelarr ./cmd/labelarr

# Runtime stage
FROM alpine:3.21

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

# Run the application
CMD ["./labelarr"] 
