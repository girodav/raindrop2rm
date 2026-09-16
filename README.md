# raindrop2rm

Tag an article `#remarkable` in Raindrop.io and it shows up on your
reMarkable tablet a few minutes later.

## Usage

Two files, no repo clone needed. `docker-compose.yml`:

```yaml
services:
  sync:
    image: ghcr.io/girodav/raindrop2rm:latest
    restart: unless-stopped
    environment:
      RAINDROP_TOKEN: ${RAINDROP_TOKEN}
      RAINDROP_TAG: ${RAINDROP_TAG:-remarkable}
      RAINDROP_ARCHIVE_TAG: ${RAINDROP_ARCHIVE_TAG:-remarkable-synced}
      REMARKABLE_FOLDER: ${REMARKABLE_FOLDER:-/Raindrop}
      POLL_INTERVAL: ${POLL_INTERVAL:-900}
    volumes:
      - ./data:/home/nonroot/.config/rmapi
```

`.env`:

```sh
RAINDROP_TOKEN=your-raindrop-test-token
```

Get the token from Raindrop.io: Settings, Integrations, "For Developers",
Create test token.

The image runs as the distroless nonroot user (UID 65532), so before
first run: `mkdir -p data && chown -R 65532:65532 data`.

Pair with the reMarkable cloud (one time, interactive, needs an active
**reMarkable Connect** subscription):

```sh
docker compose run --rm --entrypoint rmapi sync ls
```

It prints a URL and a one-time code; open the URL, log in, paste the
code back. The token is saved to `./data`.

Then:

```sh
docker compose up -d
```

## How it works

1. Polls the Raindrop.io API for raindrops tagged `#remarkable`.
2. If the link is a direct PDF (or an arXiv paper), downloads it as-is.
3. Otherwise extracts the article with
   [go-readability](https://codeberg.org/readeck/go-readability) and
   packages it as a reflowable EPUB with
   [go-epub](https://github.com/go-shiori/go-epub).
4. Uploads the result via [`rmapi`](https://github.com/ddvk/rmapi), which
   talks to reMarkable's official cloud API.
5. Swaps the `#remarkable` tag for `#remarkable-synced` so the raindrop
   isn't processed again.

Papers saved as raw PDFs render fine but aren't cropped or reformatted.
If you want that, run [paper2remarkable](https://github.com/GjjvdBurg/paper2remarkable)
separately, it uploads via `rmapi` too so it fits the same setup.

## Config (env vars)

| Var | Default | Meaning |
|---|---|---|
| `RAINDROP_TOKEN` | *(required)* | Raindrop.io test token |
| `RAINDROP_TAG` | `remarkable` | Tag that triggers a sync |
| `RAINDROP_ARCHIVE_TAG` | `remarkable-synced` | Tag applied after a successful sync |
| `REMARKABLE_FOLDER` | `/Raindrop` | Destination folder on the reMarkable |
| `POLL_INTERVAL` | `900` | Seconds between polls, `0` runs once and exits |
| `RMAPI_BIN` | `rmapi` | Path to the rmapi binary |
| `WORK_DIR` | `/tmp/raindrop2rm` | Scratch dir for generated files |

## Known limitations

- Needs a reMarkable Connect subscription (cloud API access is gated
  behind it as of 2026). Alternatives exist (self-hosted
  [rmfakecloud](https://github.com/ddvk/rmfakecloud), direct SSH push, or
  KOReader's News Downloader pointed at a Raindrop RSS feed), each with
  its own tradeoffs. Ask if you want to switch.
- Extraction quality depends on go-readability; paywalled or heavily
  JS-rendered pages won't extract well.
- Raindrop Pro's "Permanent Copy" cache isn't used yet; it could replace
  the direct fetch for Pro accounts.
- No image support in the EPUB body yet.

## Building / contributing

The image itself is just the two static Go binaries on a
[distroless](https://github.com/GoogleContainerTools/distroless) base: no
shell, no package manager, runs as nonroot. Images are built and pushed
to `ghcr.io/girodav/raindrop2rm` automatically by GitHub Actions on
every push to `main`.

Local development:

```sh
mise install
mise run build
mise run test
RAINDROP_TOKEN=... RMAPI_BIN=$(which rmapi) mise run run   # single sync pass
```

Build the image yourself instead of pulling:

```sh
mise run docker-build
```

Each published image carries a signed build provenance attestation, so
you can verify it was actually built by this repo's GitHub Actions
workflow and hasn't been tampered with in the registry:

```sh
gh attestation verify oci://ghcr.io/girodav/raindrop2rm:latest --owner girodav
```
