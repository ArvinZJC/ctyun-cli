# ctyun-cli

[![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fcore%2F*&label=release)](../../releases)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/ArvinZJC/ctyun-cli)
[![GitHub License](https://img.shields.io/github/license/ArvinZJC/ctyun-cli?label=licence)](./LICENCE)

[简体中文](./README.md) | English

> [!WARNING]
> Still under active development. Will release a stable version on... one day.

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

OpenAPI entry point: [CTyun OpenAPI docs](https://eop.ctyun.cn/ebp/ctapiDocument/index). The API documents there are the C-side API documents mentioned above.

## Highlights

- Tables by default, suitable for people, with Chinese and English display-width handling.
- `--output json` for scripts and other tools.
- Product commands supplied by plugin metadata instead of product-specific core dispatch.
- Plugins can declare methods, paths, parameters, table columns, examples, waiters, and dangerous-operation confirmation.
- i18n support for core help, errors, runtime warnings, plugin names, command descriptions, and table labels.

## Installation

Install the native `ctyun` binary with the installation scripts below. By default, the script selects the first available channel in `stable`, `beta`, then `alpha` order; set `CTYUN_INSTALL_CHANNEL` to pin a channel. If GitHub access is unreliable, replace `github.com` in the URL with `gitee.com`.

macOS, Linux, and WSL:

```sh
curl -fsSL https://github.com/ArvinZJC/ctyun-cli/releases/download/core/install.sh | bash
```

Windows PowerShell:

```powershell
irm https://github.com/ArvinZJC/ctyun-cli/releases/download/core/install.ps1 | iex
```

If you are not sure whether the current terminal is PowerShell, open Windows PowerShell from the Start menu or the Windows Terminal tab menu, then run `$PSVersionTable.PSVersion` to confirm. After it prints version information, run the installation command above in the same window.

The installation scripts support these environment variables:

| Variable                | Purpose                                                                                                                                     |
|-------------------------|---------------------------------------------------------------------------------------------------------------------------------------------|
| `CTYUN_INSTALL_CHANNEL` | Pin the install channel to `stable`, `beta`, or `alpha`                                                                                     |
| `CTYUN_INSTALL_SOURCE`  | Pin the install source to `auto`, `github`, or `gitee`                                                                                      |
| `CTYUN_INSTALL_DIR`     | Override the install directory; defaults to `$HOME/.local/bin` on macOS, Linux, and WSL, and `%LOCALAPPDATA%\Programs\ctyun-cli` on Windows |

## Plugins

A fresh `ctyun` install includes only core commands; product plugins are not preinstalled. Product commands come from plugin bundles. Install the plugins you need before running product commands:

<details>
<summary>Plugin table</summary>

| Name                 | Plugin   | Product  | Version                                                                                                                                      | Channel | Quality     | Commands | Operations |
|----------------------|----------|----------|----------------------------------------------------------------------------------------------------------------------------------------------|---------|-------------|---------:|-----------:|
| Elastic Cloud Server | `ecs`    | `ecs`    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fecs%2F*&label=release)](../../releases)    | `alpha` | `generated` |        3 |          3 |
| Region               | `region` | `region` | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fregion%2F*&label=release)](../../releases) | `alpha` | `generated` |        5 |          5 |

The quality field describes plugin metadata maturity: `generated` is a tool-generated draft, `reviewed` has passed a project review, and `curated` is kept as a maintained reference set.

</details>

```sh
ctyun plugin search ecs --source auto
ctyun plugin list --available --source auto
ctyun plugin list --available --cols Plugin,Quality,Status --filter Status=available --source auto
ctyun plugin install region ecs --source auto
ctyun plugin install --all --source auto
ctyun plugin list
```

Plugin search, available-plugin listing, install, reinstall, and update commands all support the `--source` and `--channel` options. `ctyun plugin list --available` shows hosted plugins with local installation status; `ctyun plugin search` supports fuzzy matching and follows the table/JSON output controls. `ctyun plugin reinstall` refreshes installed plugins from the selected source even when the version number has not changed. `--cols`, `--filter`, and `--sort` accept the column labels shown in the table, while stable column keys remain supported. Quote values only when the shell would split them, such as English column labels with spaces.
Dangerous operations prompt for `y/N` confirmation by default; scripts can use `--yes` or `-y` to skip the prompt.

```sh
ctyun plugin reinstall ecs region --source auto
ctyun plugin reinstall --all --source auto
ctyun plugin update --all --source auto
ctyun plugin update --all --source auto --channel alpha
ctyun plugin remove ecs region --yes
```

## Quick Start

After installing the matching plugins, common command shapes look like this:

```sh
ctyun region list
ctyun region list --name 华东1 --cols "Region ID,Region Name,Region Code"
ctyun ecs instance list --cols "Instance ID,Name,Status"
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
ctyun ecs instance list --filter Status=running --sort "-Instance ID"
```

## Authentication, Config, And Language

Config lookup order is `--config`, `CTYUN_CONFIG`, then `~/.ctyun/config.json`; `--profile` overrides `active_profile`. Apart from options that locate the config file itself, runtime settings resolve from command-line option, environment variable, active profile, then supported top-level config fallback. When the same setting exists in both an environment variable and config, the environment variable wins. `CTYUN_CONFIG` is the exception: it locates the config file, so it cannot have a fallback inside that file.

Common environment variables:

| Variable                        | Purpose                                                                                 |
|---------------------------------|-----------------------------------------------------------------------------------------|
| `CTYUN_CONFIG`                  | Override the config file path                                                           |
| `CTYUN_AK`                      | CTyun AK for live requests                                                              |
| `CTYUN_SK`                      | CTyun SK for live requests                                                              |
| `CTYUN_LANGUAGE`                | Override the interface language with `zh-CN`, `en-US`, or `en-GB`                       |
| `CTYUN_WARN_CONFIG_CREDENTIALS` | Set to `0` to disable the warning when AK/SK come from config                           |
| `CTYUN_PLUGIN_SOURCE`           | Default source for plugin install, search, and update; use `auto`, `github`, or `gitee` |
| `CTYUN_UPGRADE_SOURCE`          | Default source for core updates; use `auto`, `github`, or `gitee`                       |

Live requests prefer AK/SK from the process environment:

```sh
export CTYUN_AK=...
export CTYUN_SK=...
```

When `CTYUN_AK` or `CTYUN_SK` is missing, `ctyun` falls back to `ak`/`sk` in the active profile, then top-level config. A live command that actually uses config AK/SK writes a warning to stderr; disable it by setting the `CTYUN_WARN_CONFIG_CREDENTIALS=0` environment variable or running `ctyun config set warn_config_credentials false`.

Security recommendations:

- Prefer environment variables for AK/SK; if you store them in config, keep the file out of repositories and restrict its permissions.
- Do not store AK/SK in scripts, shell history, or logs.
- Use least-privilege IAM user AK/SK for `ctyun` and rotate them regularly.
- Avoid exposing environment variables on shared machines or in CI logs.
- When using `--debug`, inspect logs again before sharing them.

Config files can hold resource pool, language, timeout, registry, endpoint overrides for testing, and fallback values for `CTYUN_AK`/`CTYUN_SK`. The loader still rejects unsupported secret-like fields.

```json
{
  "warn_config_credentials": true,
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "cn-huadong1",
      "language": "en-GB",
      "ak": "...",
      "sk": "...",
      "timeout_seconds": 20
    }
  }
}
```

Use non-interactive commands to inspect and update config:

```sh
ctyun config path
ctyun config show
ctyun config set region cn-huadong1 --profile prod
ctyun config profile use prod
printf '%s\n' "$CTYUN_AK" | ctyun config profile set-secret prod ak --from-stdin
printf '%s\n' "$CTYUN_SK" | ctyun config profile set-secret prod sk --from-stdin
ctyun config reset --yes
```

`ctyun config show` masks saved AK/SK values like `aa*****dd`; unset values stay empty. `ctyun config reset` prompts for confirmation, then creates a backup before deleting the current config file. Scripts can use `--yes` or `-y` to skip the prompt.

Supported languages are `zh-CN`, `en-US`, and `en-GB`. Language resolution is `--lang`, then `CTYUN_LANGUAGE`, then profile `language`, then the OS locale. If nothing matches, `zh-CN` is used.

## Core Updates

Once release packages are available, use `ctyun update` or `ctyun upgrade` to check and update the core binary. Core updates only read hosted release assets from `auto`, `github`, or `gitee`; `auto` reads GitHub release assets first and falls back to the Gitee mirror. Signed indexes and SHA-256 checksums are the trust boundary. Use `--channel` to select the `stable`, `beta`, or `alpha` channel.

```sh
ctyun update --check --source auto
ctyun upgrade --source auto
ctyun upgrade --source auto --channel alpha
```

## Uninstallation

Before uninstalling the core binary, optionally remove installed plugins and config files. Plugin removal prompts for `y/N` confirmation; remove multiple plugins by name or remove every plugin. Scripts can use `--yes` or `-y` to skip the prompt:

```sh
ctyun plugin list
ctyun plugin remove ecs region
ctyun plugin remove --all --yes
```

To clean up the config file, run:

```sh
ctyun config reset
```

On macOS, Linux, and WSL, use `command -v` to locate the `ctyun` binary on the current `PATH`, then remove it. The default install path is `$HOME/.local/bin/ctyun`:

```sh
ctyun_path="$(command -v ctyun)" && rm -f "$ctyun_path"
```

Windows PowerShell installs to `%LOCALAPPDATA%\Programs\ctyun-cli\ctyun.exe` by default. If you set `CTYUN_INSTALL_DIR` during installation, use the same directory:

```powershell
$InstallDir = if ($env:CTYUN_INSTALL_DIR) { $env:CTYUN_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\ctyun-cli" }
Remove-Item -Force (Join-Path $InstallDir "ctyun.exe") -ErrorAction SilentlyContinue
```

## Developer And Contributor Workflow

If the default Go build cache is not writable, for example in a sandbox, use a repo-local cache first:

```sh
export GOCACHE="$PWD/.cache/go-build"
```

Development and debugging:

```sh
go run ./cmd/ctyun version
go run ./cmd/ctyun --version
go run ./cmd/ctyun help ecs instance list
go run ./cmd/ctyun --offline region list
go run ./cmd/ctyun --fixture region list
go run ./cmd/ctyun -O region list
go run ./cmd/ctyun --offline ecs instance list
go run ./cmd/ctyun --debug --offline ecs instance list
go run ./cmd/ctyun completion zsh
go run ./cmd/ctyun doctor network
```

`--offline`, `--fixture`, and `-O` all enable bundled plugin fixtures and do not call live CTyun APIs. This is useful for local debugging of command shape, table output, and parameter mapping. Fixture mode is intended for developer and test workflows, so all three options are omitted from regular help.

Development builds can use `--bundled` to search, list, install, reinstall, or update plugins from in-tree plugin metadata. Like `--fixture`, `--bundled` is for development and test workflows and is omitted from regular help.

```sh
go run ./cmd/ctyun plugin list --available --bundled
go run ./cmd/ctyun plugin search ecs --bundled
go run ./cmd/ctyun plugin install ecs --bundled
go run ./cmd/ctyun plugin reinstall ecs --bundled
go run ./cmd/ctyun plugin update ecs --bundled
```

Testing:

```sh
git ls-files '*.go' | xargs gofmt -w
go test ./...
go test ./internal/cli -run Completion -v
go test ./tools/plugincheck
go run ./tools/coverage
```

After plugin changes, verify according to the affected area. Lint the changed plugin first, then run the matching offline command. If the change affects generic plugin loading, command parsing, or table rendering, add the related Go tests.

```sh
go run ./cmd/ctyun plugin lint ./plugins/ecs
go run ./cmd/ctyun --offline ecs instance list

go run ./cmd/ctyun plugin lint ./plugins/region
go run ./cmd/ctyun --offline region list

go test ./tools/plugincheck
go test ./internal/cli ./internal/plugin ./internal/output
```

The OpenAPI harvest/review pipeline is a developer tool. It is not exposed as a user command and is not included in core or plugin release artifacts. The current implementation uses normalized JSON input:

```sh
go run ./tools/openapi harvest ecs --input internal/openapi/testdata/ecs-source.json
go run ./tools/openapi diff ecs
go run ./tools/openapi generate ecs
go run ./tools/openapi review ecs
```

`openapi/products/<name>/source.json` stores the latest upstream evidence, `baseline.json` advances only when a reviewed or curated plugin is promoted, and routine history lives in git. After the reviewer marks draft quality as `reviewed` or `curated`, run:

```sh
go run ./tools/openapi promote ecs
```

The release packaging tool writes core binary archives, `core-index.json`, `core-index.sig`, installation scripts, plugin archives, `index.json`, and `index.sig`. Development tests use fake HTTP sources to verify signature and download behaviour before public assets exist; real release assets serve the installation, core update, and plugin update flows above. Core install and update entrypoints use the fixed release tag `core` as a stable asset root, while plugin install and update entrypoints use the fixed release tag `plugins`; actual versions and channels are selected by the signed `core-index.json` and `index.json`. When the tool runs against an existing output directory, it preserves existing entries for other channels, replaces the rebuilt core channel or plugin name/channel assets, and signs the merged indexes again; if the same core version is being completed with more platform archives, those platform assets are merged. SemVer tags or release pages can still be created separately for user-facing changelogs.

Core and plugin versions must follow Semantic Versioning 2.0.0. Do not prefix release versions with `v`. Use `0.1.0-alpha.1` on the `alpha` channel for the first pre-release; the defaults in `internal/version/version.go` only identify unpackaged development builds, and release packaging overrides the actual version and channel.

Developer and test environment variables:

| Variable                    | Purpose                                                                                                                    |
|-----------------------------|----------------------------------------------------------------------------------------------------------------------------|
| `CTYUN_INSTALL_BASE_URL`    | Override the release root read by installation scripts for local or temporary release-asset validation                     |
| `CTYUN_RELEASE_PRIVATE_KEY` | Private key used by the release packaging tool to sign indexes                                                             |
| `CTYUN_RELEASE_PUBLIC_KEY`  | Public key used by development builds or private distribution validation for core update and plugin index signature checks |

```sh
go run ./tools/release --generate-key
export CTYUN_RELEASE_PRIVATE_KEY="<private key from previous output>"
export CTYUN_RELEASE_PUBLIC_KEY="<public key from previous output>"
go run ./tools/release --version 0.1.0-alpha.1 --channel alpha --out ./dist/releases --platform "$(go env GOOS)/$(go env GOARCH)"
```

For real releases, GitHub remains the canonical source and CI artifact authority, while Gitee is the synchronised mirror for more reliable access from mainland China. `ctyun` trusts the signing public key and SHA-256 checksums, not the hosting platform itself.

## Related Projects

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli): another unofficial CTyun CLI, written in Python, useful as a reference for users who prefer the Python ecosystem.
