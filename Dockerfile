## Build rmapi (reMarkable cloud CLI) from the actively-maintained fork.
## Note: rmapi's go.mod has a `replace` directive, so `go install pkg@version`
## refuses it -- clone and build from source instead.
FROM golang:1.27-alpine AS rmapi-builder
ARG RMAPI_VERSION=v0.0.35
RUN apk add --no-cache git
RUN git clone --depth 1 --branch ${RMAPI_VERSION} https://github.com/ddvk/rmapi.git /src/rmapi
WORKDIR /src/rmapi
RUN go build -o /out/rmapi .

## Build raindrop2rm itself.
FROM golang:1.27-alpine AS app-builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/raindrop2rm ./cmd/raindrop2rm

## Runtime image: the two Go binaries, plus paper2remarkable for nicely
## cropped/formatted paper PDFs (arXiv, PubMed, ACM, ...).
FROM alpine:3.20
ARG P2R_VERSION=0.9.14
RUN apk add --no-cache ca-certificates tzdata \
    python3 py3-pip qpdf ghostscript poppler-utils pango \
  && pip install --no-cache-dir --break-system-packages paper2remarkable==${P2R_VERSION}
COPY --from=rmapi-builder /out/rmapi /usr/local/bin/rmapi
COPY --from=app-builder /out/raindrop2rm /usr/local/bin/raindrop2rm

# rmapi stores its device pairing here; mount a volume on this path so
# pairing survives restarts and is shared between `mise run pair` and the
# long-running sync process.
ENV RMAPI_CONFIG=/root/.config/rmapi/rmapi.conf
VOLUME /root/.config/rmapi

ENV WORK_DIR=/tmp/raindrop2rm
ENV POLL_INTERVAL=900

ENTRYPOINT ["raindrop2rm"]
