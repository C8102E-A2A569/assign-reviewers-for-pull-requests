    FROM golang:1.24-alpine AS builder

    RUN apk add --no-cache git curl
    RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz && \
        mv migrate /usr/local/bin/migrate

    WORKDIR /app

    COPY go.mod go.sum ./
    RUN go mod download

    COPY . .

    RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server cmd/server/main.go

    FROM alpine:latest

    RUN apk --no-cache add ca-certificates postgresql-client curl

    WORKDIR /root/

    COPY --from=builder /app/server .
    COPY --from=builder /usr/local/bin/migrate /usr/local/bin/migrate
    COPY --from=builder /app/migrations ./migrations

    COPY entrypoint.sh /root/entrypoint.sh
    RUN chmod +x /root/entrypoint.sh
    ENTRYPOINT ["/root/entrypoint.sh"]

    RUN chmod +x /root/entrypoint.sh

    EXPOSE 8080

    ENTRYPOINT ["/root/entrypoint.sh"]