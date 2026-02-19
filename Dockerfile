FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/lastclick ./cmd/server

RUN go install github.com/pressly/goose/v3/cmd/goose@latest

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/lastclick /usr/local/bin/lastclick
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY migrations/ /migrations/

COPY <<'EOF' /entrypoint.sh
#!/bin/sh
set -e
echo "Running migrations..."
goose -dir /migrations postgres "$DATABASE_URL" up
echo "Starting server..."
exec lastclick
EOF
RUN chmod +x /entrypoint.sh

EXPOSE 8080
ENTRYPOINT ["/entrypoint.sh"]
