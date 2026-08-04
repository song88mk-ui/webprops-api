Web Properties API
==================

This API is used to expose web configuration properties for the Frontend application.

It is a small Go HTTP service (module `s88/wp`, Go 1.23, chi router, urfave/cli
entrypoint) that serves per-client feature flags, themes, and layout configuration
from AWS S3, with a shared in-memory cache. The main consumer is the `platform-web`
Blazor frontend, which fetches this service at startup (config key `Config:BaseURL`)
and caches the response on its side for 60 minutes.

Features
--------

* Individual web properties can be created, updated, and deleted.
* Support for multiple web layouts.
* Per-client maintenance flags stored inside the same properties JSON.
* Prometheus metrics (`/metrics`) and health check (`/healthz`).

API Endpoints
-------------

* `GET /api/v1/config/{clientId}` - Get configuration for a client
* `POST /api/v1/config/{clientId}` - Update configuration for a client (requires `X-Api-Key`; optional `X-Layout` header targets the layout-specific file; body is the full config JSON)
* `GET /api/v1/properties` - Get properties for a client (requires `X-Client-Id` header; optional `X-Layout` header)
* `GET /api/v1/maintenance` - List maintenance flags for a layout (requires `X-Api-Key` and `X-Layout`)
* `POST /api/v1/maintenance` - Set maintenance flags (requires `X-Api-Key` and `X-Layout`; body: `{"clientIds": "G01,G02" or "all", "isMaintenance": true|false}`)
* `POST /api/v1/cache/flush` - Flush the whole in-memory cache (requires `X-Api-Key`)
* `DELETE /api/v1/cache/flush/{clientId}` - Evict one client from the cache (requires `X-Api-Key` and `X-Layout`)
* `GET /healthz` - Health check (returns `ok`)
* `GET /metrics` - Prometheus metrics

Auth semantics: the `X-Api-Key` middleware does not reject requests by itself; it
only marks the request as admin when the header equals the `CACHE_API_KEY` env var.
Handlers then return `403 Forbidden` (config/maintenance) or `{"status":"nok"}`
(cache endpoints) for non-admin callers. `X-Client-Id` and `X-Layout` header values
are uppercased by the middleware; the `{clientId}` URL parameter of `GET /config`
is used as-is (case-sensitive).

Example `GET /api/v1/config/{clientId}` response:

```json
{
  "ClientCode": "G01",
  "Features": {
    "BettingRules": true,
    "DarkMode": true,
    "GameTrackerProductNames": [
      "song88-light",
      "song88-dark"
    ],
    "LanguageOptions": true,
    "OGAuth": false,
    "PartialCashout": true
  },
  "Layout": {
    "AL": {
      "Themes": [
        "red",
        "gray",
        "blue",
        "orange-light",
        "orange-dark"
      ]
    },
    "EL2": {
      "Themes": [
        "red",
        "white"
      ]
    }
  }
}
```

Backend (data store)
--------------------

All properties are stored in AWS S3 Buckets: `song88-{env}-webprops` as JSON files.

* The bucket name comes from the `BUCKET_NAME` env var.
* Object keys: `{clientId}.json` at the bucket root (used by `GET /config/{clientId}`
  and `GET /properties` without `X-Layout`), or `{LAYOUT}/{CLIENTID}.json` when the
  `X-Layout` header is used (e.g. `AL/G01.json`).
* Maintenance flags are the `isMaintenance` (plus optional `clientId` / `site`)
  fields inside those same JSON objects. `POST /config` preserves the existing
  `isMaintenance` value unless the payload sets it explicitly.
* Reads are cached in-process for 24 hours (86400 seconds). Cache keys:
  `{CLIENTID}` without layout, `{LAYOUT}-{CLIENTID}` with layout. Writes evict the
  corresponding key; the `/api/v1/cache/flush` endpoints evict manually.
* There is no database and no migrations - S3 is the only data store.

Prerequisites
-------------

* Go 1.23+ (see `go.mod`) - only needed to build/run natively.
* Docker - only needed for the container build (`Dockerfile` is multi-stage,
  distroless runtime; no docker-compose file exists in this repo).
* AWS credentials with `s3:GetObject`, `s3:ListBucket`, and `s3:PutObject` on the
  target bucket. Credentials are resolved through the standard AWS SDK default
  chain (`config.LoadDefaultConfig`), so env vars, `~/.aws` profiles, or an
  instance/IRSA role all work.
* An S3 bucket (or S3-compatible endpoint) containing the properties JSON files.

Configuration
-------------

Configuration is environment variables only. A `.env` file in the working
directory is auto-loaded via godotenv (`backend/main.go`). Do not commit real
secrets; `.env` values are for local use.

| Variable | Required | Purpose |
|---|---|---|
| `PORT` | yes | HTTP listen port. No default in code; the Dockerfile sets `PORT=8000`. |
| `BUCKET_NAME` | yes | S3 bucket holding the properties JSON files. |
| `CACHE_API_KEY` | yes for admin endpoints | Shared secret compared against the `X-Api-Key` header. |
| `DEBUG` | no | `true` enables chi request logging. |
| AWS credential vars | yes (or profile/role) | Standard AWS SDK env vars, e.g. `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_REGION`. |

Example `.env` (placeholders only - never use real keys in files you might commit):

```dotenv
PORT=8000
BUCKET_NAME=song88-dev-webprops
CACHE_API_KEY=<local-dev-key>
DEBUG=true
AWS_REGION=<region>
AWS_ACCESS_KEY_ID=<your-key-id>
AWS_SECRET_ACCESS_KEY=<your-secret>
```

Running locally
---------------

Natively (requires Go):

```sh
go mod download
go run main.go web
# or build first:
go build -o bin/webprops main.go
./bin/webprops web
```

`web` is the only CLI command (start the API server). On success the log prints
`api server started on port :8000` (whatever `PORT` is set to).

With Docker:

```sh
docker build -t webprops .
docker run --rm -p 8000:8000 --env-file .env webprops
```

The image entrypoint is `/app/bin/webprops` with default command `web` and
`PORT=8000` baked in.

Smoke test:

```sh
curl http://localhost:8000/healthz
# -> ok

curl http://localhost:8000/api/v1/config/G01

curl -H "X-Client-Id: G01" -H "X-Layout: AL" \
  http://localhost:8000/api/v1/properties

# Admin call (X-Api-Key must equal CACHE_API_KEY):
curl -X POST -H "X-Api-Key: $CACHE_API_KEY" \
  http://localhost:8000/api/v1/cache/flush
```

Seeding local data
------------------

The service only reads/writes JSON objects in the bucket named by `BUCKET_NAME`.
To seed data, upload a JSON file matching the example response shape above:

```sh
# Root-level file, served by GET /api/v1/config/G01
aws s3 cp G01.json s3://<your-bucket>/G01.json

# Layout-specific file, served by GET /api/v1/properties with X-Layout: AL
aws s3 cp G01.json s3://<your-bucket>/AL/G01.json
```

Alternatively, seed through the API itself with `POST /api/v1/config/{clientId}`
(admin, full-replace of the object). To avoid touching real AWS buckets, point the
AWS SDK at a local S3-compatible endpoint (e.g. LocalStack or MinIO) using the
SDK's standard endpoint configuration - the code itself has no custom endpoint
handling, it relies entirely on `config.LoadDefaultConfig`.

Remember the 24-hour read cache: after editing a file directly in S3, either
restart the server or call `POST /api/v1/cache/flush` to see the change.

Running tests
-------------

There are currently no automated tests in this repository (no `*_test.go` files),
so `go test ./...` finds nothing to run. Validation is done by:

1. Building: `go build ./...` (or `docker build -t webprops .` when no local Go
   toolchain is available - the CI workflow in `.github/workflows/ci.yml` performs
   the same Docker build on every push/PR to `main`).
2. Running the server locally and exercising the endpoints with the smoke-test
   curl commands above.

If you add tests, name them `*_test.go` next to the code under `backend/` and run
them with `go test ./...`.

## Keeping this README up to date

This README is part of the codebase. Update it in the same PR whenever anything it documents changes - project structure (projects/services added, renamed, or removed), configuration keys, run commands, ports, endpoints, or external dependencies. An outdated run guide is worse than none.
