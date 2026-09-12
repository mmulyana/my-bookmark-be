# ---- Build stage ----
FROM golang:1.26-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/server ./cmd/server

# ---- Runtime stage ----
FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/bin/server ./server
COPY --from=builder /app/db/migrations ./db/migrations

EXPOSE 8080
CMD ["./server"]
