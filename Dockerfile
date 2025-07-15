# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates make gcc musl-dev sqlite-dev

COPY go.* ./
RUN go mod tidy
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -a -o main ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /app/.env .

# SQLite dosyası için data klasörü
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./main"] 