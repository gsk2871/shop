FROM golang:1.25-bookworm AS build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./cmd/server
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/* \
    && useradd -u 10001 -m shop
USER shop
WORKDIR /app
COPY --from=build /out/server /app/server
ENTRYPOINT ["/app/server", "-conf", "/etc/shop/config.yaml"]
