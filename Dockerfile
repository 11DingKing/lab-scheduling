FROM golang:1.26-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o /out/lab-scheduling ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /out/lab-scheduling /app/lab-scheduling
COPY migrations /app/migrations
ENV HTTP_ADDR=:8080 DB_PATH=/data/lab.db MIGRATIONS_PATH=/app/migrations SESSION_TTL=8h
RUN mkdir -p /data
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD wget -q -O - http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/lab-scheduling"]
