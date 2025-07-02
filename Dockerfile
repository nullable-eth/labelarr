# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy source code (includes go.mod)
COPY . .
RUN go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o labelarr ./cmd/labelarr

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests and debugging tools
RUN apk update && apk upgrade && \
     apk add --no-cache ca-certificates tzdata bash curl wget busybox-extras && \
     ln -sf /bin/bash /bin/sh && \
     echo "Bash installed successfully" && \
     which bash && bash --version

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/labelarr .

# Create a non-root user
RUN adduser -D -s /bin/bash labelarr
USER labelarr

# Run the application
CMD ["./labelarr"] 