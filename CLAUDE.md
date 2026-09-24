# Web Properties API (webprops-api)

Small Go HTTP API that serves web configuration properties (feature flags, themes, layouts) to the Song88 frontend applications, and manages per-client maintenance flags. Module name is `s88/wp`, Go 1.23.

For the full endpoint/env/storage/deploy reference, see `.claude/skills/webprops-api/SKILL.md`.

## Commands

There is NO Go toolchain installed on this Linux dev machine (checked 2026-09-24). Options:

- Build via Docker: `docker build -t webprops .` (multi-stage, distroless runtime)
- CI builds on every push/PR to main (`.github/workflows/ci.yml`)
- If Go is available: `go build -o bin/webprops main.go`, run with `go run main.go web`

The binary is a urfave/cli app; the only command is `web` (start the server). The only tests are in `backend/server/errors_test.go` (`go test ./...`; CI runs them on pull requests only). There is no linter config.

Local config comes from a `.env` file (godotenv autoload in `backend/main.go`). Required env: `PORT`, `BUCKET_NAME`, `CACHE_API_KEY`; optional: `DEBUG=true` (request logging), plus standard AWS credential env vars.

## Architecture

- `main.go` - CLI entrypoint, registers `backend.Command`
- `backend/main.go` - `web` command: creates server, sets 2MB JSON-only body decoder, mounts routes, ListenAndServe with interrupt handling
- `backend/routes.go` - mounts everything: `/api/v1/{properties,maintenance,cache}`, `/healthz`, `/metrics` (Prometheus)
- `backend/server/` - shared plumbing: chi router + middleware (`server.go`), `APIError`/`WrapHandler` error rendering (`errors.go`), context middlewares `MustClient` (X-Client-Id), `MaybeLayout` (X-Layout), `MustApiKey` (X-Api-Key)
- `backend/routes/v1/properties/` - GET config/properties and admin POST config (full-replace update), backed by AWS S3 bucket `BUCKET_NAME` (`song88-{env}-webprops`)
- `backend/routes/v1/maintenance/` - GET/POST maintenance flags, stored as fields inside the same S3 properties JSON (`{LAYOUT}/{CLIENTID}.json`)
- `backend/routes/v1/admin/` - cache flush endpoints (`POST /cache/flush`, `DELETE /cache/flush/{clientId}`), gated by `MustApiKey`
- `backend/log/` - global zap sugared logger, use `log.Infof` etc.

Handler pattern: each route package embeds `*server.Server` in a local `Server` struct, exposes `Mount(srv, router)`, and handlers have signature `func(w, r) error` wrapped with `server.WrapHandler`. Return `server.APIError{Code: ...}` for client-visible errors; any other error becomes a logged 500.

## Critical gotchas

- `cache.NewCacher(cache.Options{Adapter: "memory"})` is called per request. This works because code.forgejo.org/go-chi/cache registers ONE `MemoryCacher` singleton in its `init()` and `NewCacher` always returns that same instance. All handlers share one in-memory store; do not "fix" the per-request call into per-request state assumptions.
- `MustApiKey` does NOT reject unauthenticated requests. It only sets an admin flag in context when `X-Api-Key` matches `CACHE_API_KEY`; each handler must check `server.AdminFromContext(ctx)` itself and respond accordingly.
- Single storage backend: everything lives in AWS S3 (maintenance was migrated off Azure Blob Storage in 2026-07). POST /maintenance rewrites `isMaintenance` inside the S3 properties JSON and evicts the `{LAYOUT}-{CLIENTID}` cache key, so the next GET /properties serves the new flag.
- Cache keys: `{clientId}` for no layout, `{LAYOUT}-{CLIENTID}` with layout. `MustClient` and `MaybeLayout` uppercase their header values, but `getConfig` takes clientId from the URL without uppercasing, so S3 keys for /config are case-sensitive to the caller.
- `get_config.go` and `get_properties.go` are near-duplicates by design of the current code; a change to one usually needs mirroring in the other.
- `server.Tag` is "latest" and is not wired to any ldflags; the CI `VERSION` build-arg is passed but the Dockerfile has no `ARG VERSION`, so version stamping is currently a no-op.

## Conventions

- Error text/codes: add new API error codes (4xxx range) to the three maps in `backend/server/errors.go`.
- Commit messages use `[TICKET-ID] Summary` (e.g. `[DEVOPS-295] Remove Confluence reporting from CI`); older commits used a lowercase `[scope]` prefix.
- Deploys: pushing to main builds an image tagged `sha-<short>-dev` to the dev ECR (account 172065883315, ap-south-1); pull requests build without pushing. Releases tagged `*-stg` go to staging (account 172065883315, ap-south-1), `*-prd` to production (account 525375333516, ap-east-1). Image repo: `platform-webprops-api`.
