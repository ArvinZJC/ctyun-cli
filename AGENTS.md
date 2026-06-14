# AGENTS.md

## Project Shape
- This is an unofficial CTyun CLI written in Go. The shipped command name is `ctyun`; do not rename it to `ctyun-cli` in examples, binaries, or help text.
- `cmd/ctyun/main.go` is intentionally tiny. Most behaviour enters through `internal/cli.Run`/`Execute`, which parses global flags, loads config/profile state, resolves language, loads plugin bundles, and dispatches either core commands or metadata-defined product commands.
- Product commands must come from plugin bundle metadata, not hardcoded product branches in `internal/cli/cli.go`. Existing examples are `plugins/ecs` and `plugins/region`; they define `plugin.json`, `commands.json`, `apis.json`, `tables.json`, `waiters.json`, fixtures, and `i18n/*.json`.
- Live API execution flows through `internal/client` and `internal/signing`: command metadata resolves profile values, path args, and flags into request query/body/header fields, then signs with CTyun EOP headers.
- `internal/output` owns stable-key table rendering and raw JSON output. Use stable column keys such as `instance_id` or `region_id`; do not use localized labels or raw CTyun field names for `--cols`, `--filter`, or `--sort`.
- `internal/registry` and `internal/plugin/install.go` own plugin install/update safety: safe plugin names, compatibility checks, `.tar.gz` extraction, checksum/signature validation, and traversal/symlink rejection.

## Workflows
- Baseline check: `GOCACHE=.cache/go-build go test ./...`. The repo-local cache avoids sandbox/cache permission failures and is already ignored.
- Coverage gate: `GOCACHE=.cache/go-build go run ./tools/coverage`. It writes under `.cache/coverage`, filters the known wrapper/platform blocks in `internal/coverprofile`, and expects total coverage to be `100.0%`.
- Useful smoke commands: `go run ./cmd/ctyun version`, `go run ./cmd/ctyun --offline ecs instance list`, and `go run ./cmd/ctyun --offline region list`.
- Lint bundle metadata with `go run ./cmd/ctyun plugin lint ./plugins/ecs` or `./plugins/region` after changing any plugin JSON.
- Keep public usage and installation wording in `README.md`; keep agent-only implementation and verification details here.

## Config, Credentials, And Live Calls
- Credentials are process-only: `CTYUN_AK` and `CTYUN_SK`. `internal/config.Load` rejects persisted AK/SK or secret material in config files.
- Config precedence is `--config`, embedded `ConfigPath`, `CTYUN_CONFIG`, then `~/.ctyun/config.json`; `--profile` overrides `active_profile`.
- Registry precedence is `--registry`, `CTYUN_REGISTRY_URL`, then profile `registry.url`/`registry_url`; HTTP registries require `index.sig` plus `CTYUN_REGISTRY_PUBLIC_KEY` or profile public key.
- Live verification should stay on safe retrieval paths such as `region list` or ECS list/show, using ephemeral credentials. Use `--offline` or `--fixture` when testing bundled fixtures.
- `--debug` writes redacted HTTP diagnostics to stderr; preserve the existing redaction path for AK/SK, request IDs, and signatures.

## Plugin Conventions
- Add product coverage by adding/reviewing plugin metadata and tests, not by adding `case "ecs"`-style dispatch. Command paths may use simple words and `{argument}` placeholders.
- `plugin.json` requires `stable|beta|edge` channels, `generated|reviewed|curated` quality, CTyun version constraints, product metadata, and an HTTPS `api.endpoint_url` when live execution is supported.
- `commands.json` binds a command ID/path to an operation, table, optional flags, fixture, docs URL, examples, dangerous confirmation, and aliases. Dangerous commands require `--yes`.
- `apis.json` maps `$profile.region`, `$arg.<name>`, and `$param.<name>` into request fields. Retrieval operations may set `retryable`; state-changing operations should not unless metadata explicitly opts in.
- `tables.json` defines display row paths and stable column keys. If a command flag maps to a table column target, fixture output is locally filtered the same way the live request is parameterised.
- Add localized help text under plugin `i18n/*.json` and table labels under `tables.json`; raw CTyun JSON payloads should remain unchanged.
