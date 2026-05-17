FROM golang:1.25-alpine AS builder-base

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

FROM builder-base AS builder-api

ARG SERVICE=api
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/app ./cmd/${SERVICE}

RUN wget -O /out/migrate.tar.gz https://github.com/golang-migrate/migrate/releases/download/v4.18.2/migrate.linux-amd64.tar.gz \
    && tar -xzf /out/migrate.tar.gz -C /out \
    && chmod +x /out/migrate

FROM builder-base AS builder-proxy

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/app ./cmd/proxy

FROM alpine:3.22 AS api

RUN apk --no-cache add ca-certificates
RUN adduser -D -H appuser
WORKDIR /app
COPY --from=builder-api /out/app /app/app
COPY --from=builder-api /out/migrate /app/migrate
COPY config /app/configs
COPY migrations /app/migrations
COPY scripts/entrypoint.sh /app/entrypoint.sh

RUN chmod +x entrypoint.sh
RUN chown -R appuser /app
USER appuser

EXPOSE 8080
ENTRYPOINT ["/app/entrypoint.sh"]

FROM alpine:3.22 AS proxy

RUN apk add --no-cache ca-certificates tzdata
RUN adduser -D -H appuser
WORKDIR /app
COPY --from=builder-proxy /out/app /app/app
COPY config /app/configs
RUN chown -R appuser /app
USER appuser

EXPOSE 8081
ENTRYPOINT ["/app/app"]