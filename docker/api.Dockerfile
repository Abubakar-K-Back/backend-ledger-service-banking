# docker/api.Dockerfile
FROM golang:1.20-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o api-server ./cmd/api

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app


COPY --from=builder /app/api-server .

EXPOSE 8080

#running the application
CMD ["./api-server"]