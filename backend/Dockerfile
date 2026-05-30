# Build stage
FROM golang:1.25.6-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN go build -ldflags '-X main.version=0.1.0' -o server .

# Runtime stage
FROM alpine:3.20

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/main.go .

EXPOSE 3001

CMD ["./server"]