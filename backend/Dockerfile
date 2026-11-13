# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS build
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=secret,id=proxy_ca --mount=type=cache,target=/go/pkg/mod \
    if [ -f /run/secrets/proxy_ca ]; then export SSL_CERT_FILE=/run/secrets/proxy_ca; fi; \
    go mod download
COPY backend/ ./
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -mod=readonly -o /server .
FROM alpine:3.21
RUN --mount=type=secret,id=proxy_ca \
    if [ -f /run/secrets/proxy_ca ]; then export SSL_CERT_FILE=/run/secrets/proxy_ca; fi; \
    apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /server /app/server
COPY db/ /app/db/
ENV LANGUAGE_MIGRATION_PATH=/app/db/002_language_pairs.sql
ENV DAILY_QUESTS_MIGRATION_PATH=/app/db/003_daily_quests.sql
ENV LITERATURE_MIGRATION_PATH=/app/db/004_literature_ai.sql
ENV PORT=8080 MIGRATIONS_PATH=/app/db/001_init.sql
EXPOSE 8080
RUN chmod 0755 /app /app/db && chmod 0644 /app/db/*.sql
USER 65532:65532
CMD ["/app/server"]
