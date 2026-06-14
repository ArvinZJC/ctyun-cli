# ctyun-cli

![GitHub commit activity](https://img.shields.io/github/commit-activity/m/ArvinZJC/ctyun-cli)
[![GitHub](https://img.shields.io/github/license/ArvinZJC/ctyun-cli)](./LICENCE)

[简体中文](./README.md) | English

`ctyun-cli` is the repository name. `ctyun` is the command-line tool name. This is an unofficial CLI for CTyun, written in Go and built on top of CTyun OpenAPI. It is plugin-based, user-experience-first, and intended for querying and managing CTyun resources from the terminal. CTyun does not currently provide an official CLI.

CTyun's official Go SDK is named `ctyun-go-sdk`, but it has limited product coverage and is not publicly released. Users who need the official SDK can submit a CTyun work order. This project is not an SDK; it is a command-line tool for user workflows.

## Before You Use It

- Activate the CTyun service you want to operate before using the CLI.
- Make sure you understand the corresponding OpenAPI.
- This project is based on CTyun OpenAPI C-side APIs, meaning consumer/customer-side APIs.
- It does not support B-side APIs, meaning business/operations-side APIs.
- It supports first-class nodes, meaning self-operated resource pools.
- It does not support second-class nodes, meaning joint-operation pools.
- Only AK/SK authentication is supported because CTyun OpenAPI currently only supports AK/SK.
- Distribution packages are not available yet; run from source for now.

OpenAPI entry point: [CTyun OpenAPI docs](https://eop.ctyun.cn/ebp/ctapiDocument/index). The API documents there are the C-side API documents mentioned above.

## Highlights

- Tables by default, suitable for people, with Chinese and English display-width handling.
- `--output json` for scripts and other tools.
- Product commands supplied by plugin metadata instead of product-specific core dispatch.
- Plugins can declare methods, paths, parameters, table columns, examples, waiters, and dangerous-operation confirmation.
- i18n support for core help, errors, plugin names, command descriptions, and table labels.

## Quick Start

Run from source:

```sh
go run ./cmd/ctyun version
go run ./cmd/ctyun help
go run ./cmd/ctyun --offline region list
go run ./cmd/ctyun --offline ecs instance list
```

When installed as `ctyun`, common command shapes look like this:

```sh
ctyun region list
ctyun region list --name 华东1 --cols region_id,region_name,region_code
ctyun ecs instance list --cols instance_id,name,status
ctyun ecs instance show ins-demo-1
ctyun --yes ecs instance start ins-demo-1
ctyun --wait ecs.instance.running ecs instance show ins-demo-1
```

Output controls:

```sh
ctyun ecs instance list --output json
ctyun ecs instance list --table compact
ctyun ecs instance list --table plain
ctyun ecs instance list --no-header
ctyun ecs instance list --filter status=running --sort -instance_id
```

## Authentication, Config, And Language

Live requests read AK/SK only from the process environment:

```sh
export CTYUN_AK=...
export CTYUN_SK=...
```

Security recommendations:

- Do not store AK/SK in repositories, scripts, config files, shell history, or logs.
- Use least-privilege IAM user AK/SK for `ctyun` and rotate them regularly.
- Avoid exposing environment variables on shared machines or in CI logs.
- When using `--debug`, inspect logs again before sharing them.

Config files are for non-secret defaults such as resource pool, language, timeout, registry, or endpoint overrides for testing. The loader rejects fields that look like AK/SK or secret material.

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

Config lookup order is `--config`, `CTYUN_CONFIG`, then `~/.ctyun/config.json`. Use `--profile` to select a named profile.

Supported languages are `zh-CN`, `en-US`, and `en-GB`. Language resolution is `--lang`, then `CTYUN_LANGUAGE`, then profile `language`, then the OS locale. If nothing matches, `zh-CN` is used.

## Plugins

Product commands come from plugin bundles. ECS and Region/resource-pool queries
are currently supported through plugins. Their bundles live in `plugins/ecs` and
`plugins/region`, and are still under active development.

```sh
ctyun plugin lint ./plugins/ecs
ctyun plugin install ./plugins/ecs
ctyun plugin install ./ctyun-plugin-ecs-0.4.2.tar.gz
ctyun plugin list
ctyun plugin remove ecs
```

Registries can be local directories or signed HTTP(S) indexes. Resolution order is `--registry`, `CTYUN_REGISTRY_URL`, then the active profile's registry configuration.

```sh
ctyun plugin search --registry ./registry
ctyun plugin install ecs --registry ./registry
ctyun plugin update --all --registry ./registry
```

## Developer And Contributor Workflow

If the default Go build cache is not writable, for example in a sandbox, use a
repo-local cache first:

```sh
export GOCACHE="$PWD/.cache/go-build"
```

Development and debugging:

```sh
go run ./cmd/ctyun version
go run ./cmd/ctyun help ecs instance list
go run ./cmd/ctyun --offline region list
go run ./cmd/ctyun --offline ecs instance list
go run ./cmd/ctyun --debug --offline ecs instance list
go run ./cmd/ctyun completion zsh
go run ./cmd/ctyun doctor network
```

`--offline` uses plugin fixtures and does not call live CTyun APIs. It is useful for local debugging of command shape, table output, and parameter mapping.

Testing:

```sh
GOCACHE="$PWD/.cache/go-build" go test ./...
GOCACHE="$PWD/.cache/go-build" go test ./internal/cli -run TestCompletion -v
GOCACHE="$PWD/.cache/go-build" go run ./tools/coverage
```

After plugin changes, verify according to the affected area. Lint the changed
plugin first, then run the matching offline command. If the change affects
generic plugin loading, command parsing, or table rendering, add the related Go
tests.

```sh
go run ./cmd/ctyun plugin lint ./plugins/ecs
go run ./cmd/ctyun --offline ecs instance list

go run ./cmd/ctyun plugin lint ./plugins/region
go run ./cmd/ctyun --offline region list

GOCACHE="$PWD/.cache/go-build" go test ./internal/cli ./internal/plugin ./internal/output
```

## Related Projects

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli): another unofficial CTyun CLI, written in Python, useful as a reference for users who prefer the Python ecosystem.
