FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make nodejs npm

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build frontend
WORKDIR /app/web
RUN npm install && npm run build

# Build backend
WORKDIR /app
RUN make build

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create app user
RUN addgroup -g 1000 gauge && \
    adduser -D -u 1000 -G gauge gauge

WORKDIR /home/gauge

# Copy binary from builder
COPY --from=builder /app/build/html-report-enhanced /usr/local/bin/
COPY --from=builder /app/web/themes /home/gauge/themes
COPY --from=builder /app/plugin.json /home/gauge/

# Set ownership
RUN chown -R gauge:gauge /home/gauge

# Switch to non-root user
USER gauge

# Expose port for serve command
EXPOSE 8080

# Default command
ENTRYPOINT ["html-report-enhanced"]
CMD ["--help"]