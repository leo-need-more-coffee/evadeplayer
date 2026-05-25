FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod ./
RUN go mod download -x 2>/dev/null; true
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -buildvcs=false -ldflags="-s -w" -o /transcoder ./cmd/transcoder

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata ffmpeg
WORKDIR /app
COPY --from=builder /transcoder /app/transcoder
EXPOSE 0
ENTRYPOINT ["/app/transcoder"]
