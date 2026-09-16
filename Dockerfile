## Build rmapi (reMarkable cloud CLI) from the actively-maintained fork.
## Note: rmapi's go.mod has a `replace` directive, so `go install pkg@version`
## refuses it -- clone and build from source instead.
FROM golang:1.27-alpine AS rmapi-builder
ARG RMAPI_VERSION=v0.0.35
RUN apk add --no-cache git
RUN git clone --depth 1 --branch ${RMAPI_VERSION} https://github.com/ddvk/rmapi.git /src/rmapi
WORKDIR /src/rmapi
RUN CGO_ENABLED=0 go build -o /out/rmapi .

## Build raindrop2rm itself.
FROM golang:1.27-alpine AS app-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/raindrop2rm ./cmd/raindrop2rm

## Runtime image: just the two static binaries + CA certs, run as a
## non-root user. UID 1000 matches the default first user on most single-
## user Linux boxes, so a bind-mounted ./data directory just works without
## needing to chown/chmod it for the container.
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata \
  && adduser -D -u 1000 appuser \
  && mkdir -p /home/appuser/.config/rmapi \
  && chown -R appuser:appuser /home/appuser
COPY --from=rmapi-builder /out/rmapi /usr/local/bin/rmapi
COPY --from=app-builder /out/raindrop2rm /usr/local/bin/raindrop2rm

USER appuser

# rmapi stores its device pairing here; mount a volume on this path so
# pairing survives restarts and is shared between `mise run pair` and the
# long-running sync process.
ENV RMAPI_CONFIG=/home/appuser/.config/rmapi/rmapi.conf
VOLUME /home/appuser/.config/rmapi

ENV WORK_DIR=/tmp/raindrop2rm
ENV POLL_INTERVAL=900

ENTRYPOINT ["raindrop2rm"]
