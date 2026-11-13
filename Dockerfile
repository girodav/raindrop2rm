## Build rmapi (reMarkable cloud CLI) from the actively-maintained fork.
## Note: rmapi's go.mod has a `replace` directive, so `go install pkg@version`
## refuses it -- clone and build from source instead.
FROM golang:1.27-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS rmapi-builder
ARG RMAPI_VERSION=v0.0.35
RUN apk add --no-cache git
RUN git clone --depth 1 --branch ${RMAPI_VERSION} https://github.com/ddvk/rmapi.git /src/rmapi
WORKDIR /src/rmapi
RUN CGO_ENABLED=0 go build -o /out/rmapi .

## Build raindrop2rm itself.
FROM golang:1.27-alpine@sha256:fa3d596898d9f126af7c01d14ee7597bab3f39cccf392ef36b69fe77cd681f18 AS app-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/raindrop2rm ./cmd/raindrop2rm

## Runtime image: just the two static binaries, nothing else. No shell, no
## package manager, no OS beyond CA certs -- the smallest attack surface
## this can have. Runs as the image's built-in nonroot user
## (UID 65532, home /home/nonroot).
FROM gcr.io/distroless/static-debian13:nonroot@sha256:aafe993665bf6200db38a02c11a9351e782a84b6a45e7f33b6695638c88e3931
COPY --from=rmapi-builder /out/rmapi /usr/local/bin/rmapi
COPY --from=app-builder /out/raindrop2rm /usr/local/bin/raindrop2rm

# rmapi stores its device pairing here; mount a volume on this path so
# pairing survives restarts and is shared between `mise run pair` and the
# long-running sync process.
ENV RMAPI_CONFIG=/home/nonroot/.config/rmapi/rmapi.conf
VOLUME /home/nonroot/.config/rmapi

ENV WORK_DIR=/tmp/raindrop2rm
ENV POLL_INTERVAL=900

ENTRYPOINT ["raindrop2rm"]
