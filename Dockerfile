FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /alertmanager-alert-sync ./cmd/alertmanager-alert-sync

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /alertmanager-alert-sync ./

# Default environment variables
ENV ALERTMANAGER_HOST=localhost:9093
ENV PORT=8080

EXPOSE 8080

CMD ["./alertmanager-alert-sync"]
