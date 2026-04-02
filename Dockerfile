# Multi-stage build for the augur-extender Go binary.

# --- Build stage ---
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /augur-extender ./cmd/augur-extender

# --- Runtime stage ---
FROM alpine:3.19

RUN apk add --no-cache ca-certificates
COPY --from=builder /augur-extender /usr/local/bin/augur-extender

EXPOSE 8888
ENTRYPOINT ["augur-extender"]
