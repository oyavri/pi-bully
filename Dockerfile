FROM golang:1.26.2 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o node ./cmd/node

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache curl python3 ffmpeg

COPY --from=builder /app/node .

EXPOSE 50051 8080 7946

ENTRYPOINT ["./node"]
