FROM golang:1.27.1-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -tags musl \
    -ldflags="-w -s -extldflags '-static'" \
    -o /app/app-bin \
    ./cmd/main.go

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S app -G app

WORKDIR /app
COPY --from=builder /app/app-bin /app/app-bin
COPY --from=builder /app/config  /app/config

# Selects config/config.yaml, relative to WORKDIR (viper does not accept an absolute path here).
ENV CONFIG_PATH=config/config

EXPOSE 3000
USER app

ENTRYPOINT ["/app/app-bin"]
