# --- build -------------------------------------------------------------------
FROM golang:1.24-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
      -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -trimpath -o /out/seed ./cmd/seed
RUN CGO_ENABLED=0 go build -trimpath -o /out/contentlint ./cmd/contentlint

# --- runtime -----------------------------------------------------------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 10001 lodestar
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/seed /usr/local/bin/seed
COPY --from=build /out/contentlint /usr/local/bin/contentlint
COPY content /srv/content
USER lodestar
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/api"]
