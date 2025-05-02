FROM golang:1.24-alpine AS builder

WORKDIR /app

ENV GOCACHE=/go/std/cache
RUN go build -v std

COPY go.mod go.sum ./
RUN go mod download

COPY internal ./internal
COPY pkg ./pkg
COPY protos ./protos
COPY main.go ./main.go
RUN go build -v -o ./main .

COPY config ./config

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/config /app/config
COPY --from=builder /app/main /app/main

EXPOSE 15000
CMD ["/app/main"]