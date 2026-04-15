FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/core-api ./core-api/cmd

FROM alpine:3.20 AS production
RUN apk add --no-cache ca-certificates wget

WORKDIR /app
COPY --from=builder /bin/core-api /usr/local/bin/core-api

ENTRYPOINT ["/usr/local/bin/core-api"]
