FROM golang:1.23-alpine AS build
WORKDIR /app
COPY backend/go.mod ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -mod=mod -o /server .
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /server /app/server
COPY db/001_init.sql /app/db/001_init.sql
COPY db/002_language_pairs.sql /app/db/002_language_pairs.sql
COPY db/003_daily_quests.sql /app/db/003_daily_quests.sql
COPY db/004_literature_ai.sql /app/db/004_literature_ai.sql
ENV LANGUAGE_MIGRATION_PATH=/app/db/002_language_pairs.sql
ENV DAILY_QUESTS_MIGRATION_PATH=/app/db/003_daily_quests.sql
ENV LITERATURE_MIGRATION_PATH=/app/db/004_literature_ai.sql
ENV PORT=8080 MIGRATIONS_PATH=/app/db/001_init.sql
EXPOSE 8080
CMD ["/app/server"]
