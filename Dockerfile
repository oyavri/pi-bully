FROM golang:1.26.2-alpine3.23 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o node ./cmd/node

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache curl

COPY --from=builder /app/node .

EXPOSE 50051 8080 7946

ENTRYPOINT ["./node"]
