# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_DATE=unknown

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 \
    GOOS=${TARGETOS} \
    GOARCH=${TARGETARCH} \
    go build \
    -trimpath \
    -ldflags="-s -w \
    -X main.version=${VERSION} \
    -X main.commit=${COMMIT} \
    -X main.date=${BUILD_DATE}" \
    -o /out/turnino \
    ./cmd/turnino

FROM alpine:3.24

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S turnino && \
    adduser -S -G turnino turnino && \
    mkdir -p /data && \
    chown turnino:turnino /data

COPY --from=build /out/turnino /usr/local/bin/turnino

USER turnino

VOLUME ["/data"]
EXPOSE 8080

ENV TURNINO_ADDR=:8080 \
    TURNINO_DATA_DIR=/data

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O - http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/turnino"]
