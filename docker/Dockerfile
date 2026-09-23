# Production image.
#
# One image carries the API, the content importer, the content linter, goose
# and the migration files. Deploying therefore ships exactly one artifact, and
# the migrations that run against the database are provably the ones that were
# built alongside the binary — not whatever happens to be checked out on the
# server.

# --- build -------------------------------------------------------------------
FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum* ./
RUN go mod download

# goose is pinned; a floating migration tool in a release image is a bad idea.
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.24.1

COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
      -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -o /out/seed        ./cmd/seed
RUN CGO_ENABLED=0 go build -trimpath -o /out/contentlint ./cmd/contentlint

# --- runtime -----------------------------------------------------------------
FROM alpine:3.24

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown
LABEL org.opencontainers.image.title="hefesto-api" \
      org.opencontainers.image.source="https://github.com/jaelricco/hefesto" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}" \
      org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.licenses="UNLICENSED"

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 hefesto

COPY --from=build /out/api            /usr/local/bin/api
COPY --from=build /out/seed           /usr/local/bin/seed
COPY --from=build /out/contentlint    /usr/local/bin/contentlint
COPY --from=build /go/bin/goose       /usr/local/bin/goose
COPY db/migrations /srv/migrations
COPY content       /srv/content

ENV GOOSE_DRIVER=postgres \
    GOOSE_MIGRATION_DIR=/srv/migrations \
    HEFESTO_CONTENT_DIR=/srv/content

USER hefesto
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=20s --retries=4 \
    CMD ["/usr/local/bin/api", "-healthcheck"]
ENTRYPOINT ["/usr/local/bin/api"]
