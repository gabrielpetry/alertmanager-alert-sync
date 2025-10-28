from golang:1.25-alpine as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go ./
RUN go build -o /alertmanager_sync

FROM alpine:latest
WORKDIR /app
COPY --from=builder /alertmanager_sync ./

ENV ALERTMANAGER_HOST=http://localhost:9093
EXPOSE 8080
CMD ["/app/alertmanager_sync"]
