# raindrop2rm

Tag an article `#remarkable` in Raindrop.io and it shows up as an EPUB on
your reMarkable tablet a few minutes later.

## How it works

1. Polls the Raindrop.io API for raindrops tagged `#remarkable` (any
   collection).
2. Fetches each article and extracts the readable content with
   [go-readability](https://codeberg.org/readeck/go-readability).
3. Wraps it into a single-chapter EPUB with
   [go-epub](https://github.com/go-shiori/go-epub) (reflowable, so you get
   adjustable font size on the e-ink screen — unlike a PDF).
4. Uploads it to your reMarkable via [`rmapi`](https://github.com/ddvk/rmapi)
   (the actively-maintained fork; the original `juruen/rmapi` is archived).
   This talks to reMarkable's official cloud API, which requires an active
   **reMarkable Connect** subscription.
5. Swaps the `#remarkable` tag for `#remarkable-synced` on the raindrop so it
   isn't processed again.

## One-time setup

**0. Docker.** This machine didn't have it installed. Real Docker Engine
needs a one-time `sudo` install (it runs as a root daemon):

```sh
sudo apt-get update
sudo apt-get install -y ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker "$USER"
# then log out and back in (or `newgrp docker`) for the group change to apply
```

**1. Raindrop test token** — in Raindrop.io go to Settings → Integrations →
"For Developers" → Create test token. Put it in `.env` (copy
`.env.example`).

**2. Build the image:**

```sh
mise run docker-build
```

**3. Pair with the reMarkable cloud** (interactive, one time only — needs
an active reMarkable Connect subscription; the token is saved into
`./data`, which is git-ignored and mounted into the container from then
on):

```sh
mise run pair
```

It will print a URL (`my.remarkable.com/device/desktop/connect` or
similar) and ask for a one-time code — open that URL on any device, log
into your reMarkable account, copy the code back into the terminal.

## Running

```sh
docker compose up -d
```

This runs `raindrop2rm` in a loop, polling every `POLL_INTERVAL` seconds
(default 900 = 15 min). Set `POLL_INTERVAL=0` to run a single pass and
exit (handy if you'd rather drive scheduling from an external cron).

```sh
docker compose logs -f       # tail logs
docker compose restart       # after an .env change
docker compose down          # stop it
```

## Local development (without Docker)

```sh
mise install          # pulls in Go
mise run build         # -> ./bin/raindrop2rm
mise run test
RAINDROP_TOKEN=... RMAPI_BIN=$(which rmapi) mise run run   # single sync pass
```

`RMAPI_BIN` lets you point at a locally installed/paired `rmapi` binary
instead of the one baked into the Docker image.

## Config (env vars)

| Var | Default | Meaning |
|---|---|---|
| `RAINDROP_TOKEN` | *(required)* | Raindrop.io test token |
| `RAINDROP_TAG` | `remarkable` | Tag that triggers a sync |
| `RAINDROP_ARCHIVE_TAG` | `remarkable-synced` | Tag applied after a successful sync |
| `REMARKABLE_FOLDER` | `/Raindrop` | Destination folder on the reMarkable |
| `POLL_INTERVAL` | `900` | Seconds between polls; `0` = run once and exit |
| `RMAPI_BIN` | `rmapi` | Path to the rmapi binary |
| `WORK_DIR` | `/tmp/raindrop2rm` | Scratch dir for generated EPUBs |

## Known limitations / ideas

- Requires a reMarkable Connect subscription (cloud API access is gated
  behind it as of 2026). Free alternatives exist (self-hosted
  [rmfakecloud](https://github.com/ddvk/rmfakecloud), direct SSH push, or
  KOReader's on-device News Downloader pointed at Raindrop's per-collection
  RSS feed) but each trades off reliability or adds real infrastructure —
  ask if you want to switch.
- Extraction quality depends on go-readability; paywalled or heavily
  JS-rendered pages won't extract well (no headless browser here by
  design, to keep the image small).
- Raindrop Pro's "Permanent Copy" cache isn't used yet — it already does
  server-side extraction/archiving and could replace the direct fetch for
  Pro accounts.
- No image support in the EPUB body currently (readability keeps `<img>`
  tags in the HTML, but they aren't embedded into the EPUB, so they won't
  render on-device). Worth adding if articles you care about lean on
  images.
