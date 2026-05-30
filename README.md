<div align="center">

### Fast, simple, and convenient backend for your video player.

Upload videos, transcode them to HLS, and get ready-to-use playback URLs by ID.

[Quick Start](#quick-start) · [Architecture](#architecture) · [API](#api)

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white)](https://docs.docker.com/compose/)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

</div>

---

## About

EvadePlayer is a backend for video playback.

It handles the whole pipeline:

- video upload
- ffmpeg transcoding
- HLS generation
- signed playback URLs
- secure media delivery

Frontend lives separately — use your own UI or connect a dedicated frontend repo.

---

## Features

- Simple REST API
- HLS streaming with signed URLs
- H.264 / H.265 / AV1 transcoding
- GPU acceleration (NVIDIA NVENC, VAAPI)
- Service-token auth for upload
- Configurable public or protected read access
- Preview sprites for timeline scrubbing

---

## Quick Start

```bash
git clone https://github.com/leo-need-more-coffee/evadeplayer-platform.git
cd evadeplayer
./setup.sh
```

Upload a video:

```bash
curl -X POST http://localhost/api/videos/upload \
  -H "X-Service-Key: $SERVICE_KEY" \
  -F file=@video.mp4
```

Get video info by ID:

```bash
curl http://localhost/api/videos/{id}
```

When processing is finished, the response contains `manifest_url` — pass it to your player directly.

---

## Architecture

```mermaid
flowchart LR
    A[Client / Frontend] --> B[EvadePlayer API]
    B --> C[(PostgreSQL)]
    B --> D[[Redis]]
    D --> E[Transcoder]
    E --> F[(SeaweedFS)]
    B --> G[Signed HLS URL]
    A --> H[nginx]
    H --> F
```

Flow:

1. Client uploads a video using a service key
2. API stores a record and enqueues a Redis job
3. Transcoder processes the file with ffmpeg
4. HLS files and sprites are stored in SeaweedFS
5. API returns a signed `manifest_url` for the video ID
6. nginx serves manifests and segments directly

---

## Auth

Upload (`POST /videos/upload`) always requires `X-Service-Key` header.

Read access (`GET /videos/*`) is controlled by `READ_PUBLIC`:

```env
# true  — anyone can fetch video info and manifests (default)
# false — X-Service-Key required for reads too
READ_PUBLIC=true

SERVICE_KEY=change-me
HLS_TOKEN_SECRET=change-me
```

HLS manifests and segments are always signed — URLs expire after 4 hours regardless of `READ_PUBLIC`.

---

## API

| Method | Path                      | Auth           | Description           |
| ------ | ------------------------- | -------------- | --------------------- |
| `POST` | `/videos/upload`          | Service key    | Upload video          |
| `GET`  | `/videos`                 | Public / key   | List videos           |
| `GET`  | `/videos/{id}`            | Public / key   | Video details + URL   |
| `GET`  | `/videos/{id}/status`     | Public / key   | Transcode status      |
| `GET`  | `/videos/{id}/storyboard` | Public / key   | Sprite cues for scrub |
| `GET`  | `/healthz`                | —              | Health check          |

---

## Config

| Variable              | Description                                         |
| --------------------- | --------------------------------------------------- |
| `SERVICE_KEY`         | Required for upload (and reads if `READ_PUBLIC=false`) |
| `READ_PUBLIC`         | `true` = open read access, `false` = key required   |
| `HLS_TOKEN_SECRET`    | Secret for signing HLS URLs                         |
| `PUBLIC_HOST`         | Public base URL, e.g. `https://example.com`         |
| `NGINX_PORT`          | Host port exposed by nginx                          |
| `MAX_UPLOAD_SIZE_GB`  | Max upload size in GB (default: 50)                 |
| `TRANSCODE_ACCEL`     | `cpu`, `nvidia`, or `vaapi`                         |
| `TRANSCODE_CODECS`    | e.g. `h264,h265,av1`                                |
| `TRANSCODE_QUALITIES` | e.g. `360p,720p,1080p`                              |

Run `./setup.sh` to configure interactively.

---

## License

[MIT](LICENSE)
