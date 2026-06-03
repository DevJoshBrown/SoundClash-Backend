FROM golang:1.26-alpine AS builder
RUN apk add --no-cache ffmpeg
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
RUN apk add --no-cache ffmpeg
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
