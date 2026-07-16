---
name: webprops-api
description: Full reference for the webprops-api repo (s88/wp). Use when adding or changing endpoints, touching the cache, maintenance flags, S3 storage access, env vars, or the release/deploy pipeline in this repository.
---

# webprops-api reference

Go 1.23, module `s88/wp`. chi router, urfave/cli entrypoint (`webprops web`), zap logging, Prometheus metrics. Serves frontend web configuration and maintenance flags from AWS S3, with a shared in-memory cache.

## Endpoint reference

| Method | Path | Auth / headers | Handler |
|---|---|---|---|
| GET | `/healthz` | none | `backend/routes/healthcheck.go` |
| GET | `/metrics` | none | promhttp |
| GET | `/api/v1/config/{clientId}` | none | `properties/get_config.go` |
| POST | `/api/v1/config/{clientId}` | `X-Api-Key` required, `X-Layout` optional, body: full config JSON | `properties/update_config.go` updateConfig |
| GET | `/api/v1/properties` | `X-Client-Id` required, `X-Layout` optional | `properties/get_properties.go` |
| GET | `/api/v1/maintenance` | `X-Api-Key` + `X-Layout` required | `maintenance/maintenance.go` getMaintenance |
| POST | `/api/v1/maintenance` | `X-Api-Key` + `X-Layout` required, body `{"clientIds": "a,b" or "all", "isMaintenance": bool}` | `maintenance/maintenance.go` setMaintenance |
| POST | `/api/v1/cache/flush` | `X-Api-Key` | `admin/cache.go` invalidateCache (flush all) |
| DELETE | `/api/v1/cache/flush/{clientId}` | `X-Api-Key` + `X-Layout` | `admin/cache.go` invalidateClientCache |

Auth semantics: `MustApiKey` middleware never rejects; it sets a context flag only when `X-Api-Key` equals env `CACHE_API_KEY`. Handlers check `server.AdminFromContext(ctx)` and return 403 (maintenance) or `{"status":"nok"}` with HTTP 200 (cache endpoints). `MustClient` rejects missing `X-Client-Id` with error code 4002 mapped to HTTP 404.

## Storage

- Everything lives in AWS S3, bucket from env `BUCKET_NAME` (naming: `song88-{env}-webprops`). Keys: `{clientId}.json` at root (no layout / GET config) or `{LAYOUT}/{CLIENTID}.json` (GET properties with `X-Layout`). AWS creds via default SDK chain; the deploy role needs s3:GetObject, s3:ListBucket, and s3:PutObject (maintenance and config writes).
- POST /config/{clientId} replaces the whole S3 object with the request body (full replace, not a merge), except `isMaintenance`: if the payload omits it, the current value is carried over from the existing object so a config push cannot silently clear maintenance mode. Without `X-Layout` the URL clientId is used as-is (root file, case-sensitive, matches GET /config); with `X-Layout` the clientId is uppercased (matches GET /properties). Evicts the corresponding cache key after upload.
- Maintenance flags are fields (`isMaintenance`, plus optional `clientId`/`site`) inside the same `{LAYOUT}/{CLIENTID}.json` properties objects. GET /maintenance lists the `{LAYOUT}/` prefix and reads each object; POST /maintenance get-modify-puts the object (rewrites `isMaintenance`, preserves the rest) and then evicts the `{LAYOUT}-{CLIENTID}` cache key so GET /properties serves the new flag immediately. clientIds in the POST body are trimmed and uppercased before use (S3 keys and cache keys are uppercase).
- History: maintenance used to live in Azure Blob Storage (account song88webconfig, env `CONTAINER`); migrated to S3 in 2026-07, Azure SDK removed.

## Cache

Library: `code.forgejo.org/go-chi/cache`, memory adapter. The library registers a single `MemoryCacher` in `init()` and `NewCacher` returns that same instance every time, so the per-request `cache.NewCacher(...)` calls all share one process-wide store. TTL on put: 86400 seconds. Keys: `{CLIENTID}` (config/properties without layout) and `{LAYOUT}-{CLIENTID}` (properties with layout, maintenance eviction, per-client flush).

Consequences: cached data survives until TTL, full flush, per-client flush, or a maintenance POST. Multiple pod replicas each have their own cache; invalidation endpoints only hit the replica that serves the request.

## Environment variables

| Var | Purpose |
|---|---|
| `PORT` | listen port (Dockerfile sets 8000) |
| `BUCKET_NAME` | S3 bucket with properties JSON |
| `CACHE_API_KEY` | shared secret for admin/maintenance endpoints |
| `DEBUG` | `true` enables chi request logging |
| AWS creds | standard SDK env chain (env vars, profile, IRSA/instance role) |

`.env` is autoloaded via godotenv (`backend/main.go` blank import).

## Adding an endpoint (established pattern)

1. Create or reuse a package under `backend/routes/v1/<area>/` with `type Server struct { *server.Server }` and a `Mount(srv *server.Server, r chi.Router)`.
2. Handlers: `func (s *Server) name(w http.ResponseWriter, r *http.Request) error`, registered via `server.WrapHandler(...)`; chain `server.MustApiKey` / `server.MustClient` / `server.MaybeLayout` with `.With(...)` as needed.
3. Client-visible failures: return `server.APIError{Code: ..., Message: ...}`. New reusable codes go in the 4xxx range in all three maps in `backend/server/errors.go`. Other returned errors are logged and rendered as generic 500.
4. Wire the package in `backend/routes.go` `mountRoutes`.
5. Request bodies: `render.Decode` is globally replaced by `backend/main.go decode` (JSON only, 2MB limit, error code 4001).

## Build, run, deploy

- No Go toolchain on this dev machine; build with `docker build -t webprops .` or let CI do it.
- Image: multi-stage, distroless nonroot, entrypoint `/app/bin/webprops`, default cmd `web`.
- CI (`.github/workflows/ci.yml`): push/PR to main builds and pushes to dev ECR (account 418541840646, ap-south-1) tagged `main-sha-*` (or `pr-sha-*`). GitHub release with tag suffix `-stg` pushes to 172065883315/ap-south-1; `-prd` pushes to 525375333516/ap-east-1. Repo: `platform-webprops-api`. Semver tags also produce version tags.
- Version stamping is broken/no-op: `server.Tag` is a constant "latest", CI passes `VERSION` build-arg but the Dockerfile declares no `ARG VERSION` and no ldflags are used.

## Known warts (do not silently "fix" without asking)

- `get_config.go` vs `get_properties.go` duplication; changes usually need mirroring.
- `getConfig` does not uppercase the URL clientId, while the header middlewares uppercase theirs; S3 lookups for /config are caller-case-sensitive.
- Cache admin endpoints return HTTP 200 with `{"status":"nok"}` on auth failure instead of 401/403.
- AWS config and S3 clients are created per request in every handler (works, relies on the SDK chain being cheap).
- No tests exist anywhere in the repo.

## Repo conventions

- Commit style: `[scope] lowercase summary` (e.g. `[docker] set cmd`).
- Logging via `s88/wp/backend/log` package functions (`log.Infof`, `log.Errorf`), never raw zap or fmt.
