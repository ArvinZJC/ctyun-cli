# ctyun

![GitHub commit activity](https://img.shields.io/github/commit-activity/m/ArvinZJC/ctyun-cli)
[![GitHub](https://img.shields.io/github/license/ArvinZJC/ctyun-cli)](./LICENCE)

> [!WARNING]
> Still under active development. Will release on... one day?

`ctyun` is an unofficial command-line interface for CTyun, built in Go. It aims
to be pleasant for day-to-day cloud operations: table output by default, raw
JSON when needed, CTyun-compatible request signing, and product commands loaded
from reviewed metadata bundles.

The project is still early, but the core shape is already usable. The binary is
small; product behaviour lives in plugin bundles so new CTyun services can be
added without growing product-specific dispatch code in the CLI core.

## Highlights

- Human-friendly table output with stable column keys and Chinese/English width
  handling.
- Raw `--output json` mode that preserves CTyun-like response payloads.
- CTyun EOP request signing with `CTYUN_AK` and `CTYUN_SK`.
- Metadata-driven plugins for commands, API mappings, tables, waiters, fixtures,
  examples, and localized help.
- Offline fixtures for predictable local demos and tests.
- Registry, tarball, and local plugin installation with validation, checksum
  support, and safe archive extraction.

## Quick Start

Build and run from source:

```sh
go run ./cmd/ctyun version
go run ./cmd/ctyun --offline region list
go run ./cmd/ctyun --offline ecs instance list
```

The repository currently includes reviewed slices for common region/resource
pool discovery and a small ECS instance workflow:

```sh
ctyun region list
ctyun region list --name 华东1 --cols region_id,region_name,region_code
ctyun ecs instance list --cols instance_id,name,status
ctyun ecs instance show ins-demo-1
ctyun --yes ecs instance start ins-demo-1
ctyun --wait ecs.instance.running ecs instance show ins-demo-1
```

Use `--offline` when you want bundled fixture responses instead of live CTyun
API calls:

```sh
ctyun --offline ecs instance list --filter status=running --sort -instance_id
ctyun --offline region list --output json
```

## Credentials And Profiles

Live requests read credentials only from the process environment:

```sh
export CTYUN_AK=...
export CTYUN_SK=...
```

Profile config may hold non-secret defaults such as region, language, timeout,
registry, or endpoint override settings. Do not store AK/SK values in config
files; the loader rejects secret-looking keys.

```json
{
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "cn-huadong1",
      "language": "en-GB",
      "timeout_seconds": 20
    }
  }
}
```

Config lookup order is `--config`, `CTYUN_CONFIG`, then
`~/.ctyun/config.json`. Select a profile with `--profile`.

## Plugins

Product commands come from plugin bundles. The included examples live in
`plugins/ecs` and `plugins/region`.

```sh
ctyun plugin lint ./plugins/ecs
ctyun plugin install ./plugins/ecs
ctyun plugin install ./ctyun-plugin-ecs-0.4.2.tar.gz
ctyun plugin list
ctyun plugin remove ecs
```

Registries can be local directories or signed HTTP(S) indexes. Resolution order
is `--registry`, `CTYUN_REGISTRY_URL`, then the active profile's registry
configuration.

```sh
ctyun plugin search --registry ./registry
ctyun plugin install ecs --registry ./registry
ctyun plugin update --all --registry ./registry
```

## Output

Tables are the default because they are easier to scan in operational workflows.
Use stable column keys for selection, filtering, and sorting:

```sh
ctyun ecs instance list --cols instance_id,name,status
ctyun ecs instance list --filter status=running
ctyun ecs instance list --sort -instance_id
```

Switch to raw JSON when integrating with other tools:

```sh
ctyun region list --output json
```

## Project Status

This repository currently provides the CLI foundation, CTyun signing and HTTP
transport, plugin lifecycle commands, signed registry consumption, table/JSON
rendering, i18n hooks, waiters, and two reviewed plugin slices.

Planned work is mainly catalog and release work: more reviewed CTyun product
bundles, an OpenAPI harvesting/review pipeline, hosted plugin registry mirrors,
release packaging, and release signing. Core self-upgrade is intentionally
deferred; install CLI updates through your package manager or release artifact
until that design exists.

## For Contributors

Go 1.25 or newer is required. The module keeps the `go` directive aligned with
the oldest Go release policy this project intends to support, not necessarily
the newest local toolchain.
