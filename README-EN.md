# ctyun-cli

[![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fcore%2F*&label=release)](../../releases)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/ArvinZJC/ctyun-cli)
[![GitHub License](https://img.shields.io/github/license/ArvinZJC/ctyun-cli?label=licence)](./LICENCE)

[简体中文](./README.md) | English

`ctyun-cli` is the repository name. `ctyun` is the command-line tool name. This is an unofficial CLI for CTyun, written in Go and built on top of CTyun OpenAPI. It is plugin-based, user-experience-first, and intended for querying and managing CTyun resources from the terminal. CTyun released the official `ctyun-cli` on 2 July 2026; this project is not the official CLI, but an independently maintained unofficial implementation.

CTyun's official Go SDK is named `ctyun-go-sdk`, but it has limited product coverage and is not publicly released. Users who need the official SDK can submit a CTyun work order. This project is not an SDK; it is a command-line tool for user workflows.

## Relationship To The Official CLI

The public entry point for CTyun's official `ctyun-cli` is the [official CLI docs](https://www.ctyun.cn/document/11095072). As of now, it does not have a separate official product home page. The official tool uses the `ctyun-cli` command name, while this project uses `ctyun`, so the two binaries do not conflict and can both be present in one shell environment. Both tools use `CTYUN_AK` / `CTYUN_SK` for AK/SK environment variables; if you use them side by side, remember that this credential pair is shared by both tools.

This project will keep iterating as an unofficial alternative to the official CLI. We will continue exploring and implementing capabilities expected from a modern cloud CLI, while keeping a better terminal experience, script friendliness, composable output, and maintainable extension paths in mind. Once the official CLI is robust, stable, flexible, capable, and polished enough for the same workflows, we can reassess this project's role and lifecycle.

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

| Variable                | Purpose                                                                                                                                          |
|-------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------|
| `CTYUN_INSTALL_CHANNEL` | Pin the installation channel to `stable`, `beta`, or `alpha`                                                                                     |
| `CTYUN_INSTALL_SOURCE`  | Pin the installation source to `auto`, `github`, or `gitee`                                                                                      |
| `CTYUN_INSTALL_DIR`     | Override the installation directory; defaults to `$HOME/.local/bin` on macOS, Linux, and WSL, and `%LOCALAPPDATA%\Programs\ctyun-cli` on Windows |

## Core Commands

These commands do not depend on product plugins. They are useful right after installation for checking the version, reading help, generating completion scripts, or checking network connectivity:

```sh
ctyun --version
ctyun help
ctyun help config
ctyun completion zsh
ctyun doctor local
ctyun doctor network
```

Plugin command help becomes available after installing the matching plugin, for example `ctyun help region list`.

## Authentication, Config, And Language

Config lookup order is `--config`, `CTYUN_CONFIG`, then `~/.ctyun/config.json`; `--profile` overrides `active_profile`. Apart from options that locate the config file itself, runtime settings resolve from command-line option, environment variable, active profile, then supported top-level config fallback. When the same setting exists in both an environment variable and config, the environment variable wins. `CTYUN_CONFIG` is the exception: it locates the config file, so it cannot have a fallback inside that file.

Common environment variables:

| Variable                        | Purpose                                                                                      |
|---------------------------------|----------------------------------------------------------------------------------------------|
| `CTYUN_CONFIG`                  | Override the config file path                                                                |
| `CTYUN_AK`                      | CTyun AK for live requests                                                                   |
| `CTYUN_SK`                      | CTyun SK for live requests                                                                   |
| `CTYUN_LANGUAGE`                | Override the interface language with `zh-CN`, `en-US`, or `en-GB`                            |
| `CTYUN_WARN_CONFIG_CREDENTIALS` | Set to `0` to disable the warning when AK/SK come from config                                |
| `CTYUN_WARN_DEPRECATED`         | Set to `0` to disable warnings when deprecated commands, options, or output fields are used  |
| `CTYUN_PLUGIN_SOURCE`           | Default source for plugin installation, search, and update; use `auto`, `github`, or `gitee` |
| `CTYUN_UPGRADE_SOURCE`          | Default source for core updates; use `auto`, `github`, or `gitee`                            |

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
  "warn_deprecated": true,
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "81f7728662dd11ec810800155d307d5b",
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
ctyun config explain
ctyun config explain region --output json
ctyun config set region 81f7728662dd11ec810800155d307d5b --profile prod
ctyun config profile use prod
printf '%s\n' "$CTYUN_AK" | ctyun config profile set-secret prod ak --from-stdin
printf '%s\n' "$CTYUN_SK" | ctyun config profile set-secret prod sk --from-stdin
ctyun config reset --yes
```

`ctyun config show` displays stored JSON and masks saved AK/SK values like `aa*****dd`; unset values stay empty. `ctyun config explain` instead reports effective base settings and the source that won for each value. Sensitive rows report only whether a value is configured and never reveal, mask, fingerprint, or otherwise derive AK/SK or registry public-key material.

Use `ctyun doctor local` for an offline, read-only health report covering the config file, profile selection, credential completeness and storage source, region, endpoint override syntax, installed-plugin directory, and each installed plugin bundle. It performs no DNS, HTTP, CTyun, registry, or release request and does not repair local state. The command always renders every independent finding; warnings and skipped checks exit zero, while any failed finding produces the complete report and exits one without an extra aggregate error line. Use `ctyun doctor network` separately for online source and CTyun endpoint diagnostics.

`ctyun config reset` prompts for confirmation, then creates a backup before deleting the current config file. Scripts can use `--yes` or `-y` to skip the prompt.

Supported languages are `zh-CN`, `en-US`, and `en-GB`. Language resolution is `--lang`, then `CTYUN_LANGUAGE`, then profile `language`, then the OS locale. If nothing matches, `zh-CN` is used.

## Plugins

A fresh `ctyun` installation includes only core commands; product plugins are not preinstalled. Product commands come from plugin bundles. After setting up authentication, config, and language preferences, install the plugins you need:

<details>
<summary>Plugin table</summary>

| Name                                      | Plugin            | Product           | Version                                                                                                                                               | Channel  | Quality     | Commands | Operations |
|-------------------------------------------|-------------------|-------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------|----------|-------------|---------:|-----------:|
| Application Cloud Server                  | `acs`             | `acs`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Facs%2F*&label=release)](../../releases)             | `beta`   | `generated` |        8 |          8 |
| Auto Scaling                              | `as`              | `as`              | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fas%2F*&label=release)](../../releases)              | `beta`   | `generated` |       62 |         62 |
| Cloud Backup and Recovery                 | `cbr`             | `cbr`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcbr%2F*&label=release)](../../releases)             | `beta`   | `generated` |       22 |         22 |
| Cloud Disaster Recovery                   | `cdr`             | `cdr`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcdr%2F*&label=release)](../../releases)             | `beta`   | `generated` |       31 |         31 |
| Cloud Function                            | `cf`              | `cf`              | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcf%2F*&label=release)](../../releases)              | `beta`   | `generated` |       62 |         62 |
| Cloud Assistant                           | `cloud-assistant` | `cloud-assistant` | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcloud-assistant%2F*&label=release)](../../releases) | `beta`   | `generated` |       11 |         11 |
| Common                                    | `common`          | `common`          | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcommon%2F*&label=release)](../../releases)          | `stable` | `curated`   |        1 |          1 |
| Dedicated Physical Server                 | `dps`             | `dps`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fdps%2F*&label=release)](../../releases)             | `beta`   | `generated` |       62 |         62 |
| CTyun Cloud Computer (Enterprise Edition) | `ecpc`            | `ecpc`            | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fecpc%2F*&label=release)](../../releases)            | `beta`   | `generated` |      279 |        279 |
| Elastic Cloud Server                      | `ecs`             | `ecs`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fecs%2F*&label=release)](../../releases)             | `beta`   | `generated` |      225 |        225 |
| Elastic High Performance Computing        | `ehpc`            | `ehpc`            | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fehpc%2F*&label=release)](../../releases)            | `beta`   | `generated` |       24 |         24 |
| Elastic Volume Service                    | `evs`             | `evs`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fevs%2F*&label=release)](../../releases)             | `beta`   | `generated` |       30 |         30 |
| High Performance File Storage             | `hpfs`            | `hpfs`            | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fhpfs%2F*&label=release)](../../releases)            | `beta`   | `generated` |       40 |         40 |
| Image Management Service                  | `ims`             | `ims`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fims%2F*&label=release)](../../releases)             | `beta`   | `generated` |       27 |         27 |
| Job                                       | `job`             | `job`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fjob%2F*&label=release)](../../releases)             | `stable` | `curated`   |        1 |          1 |
| OceanFS                                   | `oceanfs`         | `oceanfs`         | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Foceanfs%2F*&label=release)](../../releases)         | `beta`   | `generated` |       34 |         34 |
| Order                                     | `order`           | `order`           | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Forder%2F*&label=release)](../../releases)           | `stable` | `curated`   |        7 |          7 |
| Region                                    | `region`          | `region`          | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fregion%2F*&label=release)](../../releases)          | `stable` | `curated`   |        7 |          7 |
| Scalable File Service                     | `sfs`             | `sfs`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fsfs%2F*&label=release)](../../releases)             | `beta`   | `generated` |       56 |         56 |
| Volume Backup Service                     | `vbs`             | `vbs`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fvbs%2F*&label=release)](../../releases)             | `beta`   | `generated` |       37 |         37 |
| Zettabyte Object Storage                  | `zos`             | `zos`             | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fzos%2F*&label=release)](../../releases)             | `beta`   | `generated` |      106 |        106 |

The quality field describes plugin metadata maturity: `generated` is a tool-generated draft, `reviewed` has passed a project review, and `curated` is kept as a maintained reference set.

</details>

```sh
ctyun plugin search ecs --source auto
ctyun plugin list --available --source auto
ctyun plugin list --available --cols Plugin,Quality,Status --filter Status=available --source auto
ctyun plugin install region --source auto
ctyun plugin install ecs --source auto --channel beta
ctyun plugin install --all --source auto
ctyun plugin list
```

Plugin management commands share these behaviours:

- `ctyun plugin search`, `ctyun plugin list --available`, `ctyun plugin install`, `ctyun plugin reinstall`, and `ctyun plugin update` support `--source` and `--channel`.
- `ctyun plugin list --available` shows hosted plugins with local installation status.
- `ctyun plugin list --available` and `ctyun plugin search` inspect the `stable` channel by default, and can use `--channel all` to inspect every registry channel.
- Install, reinstall, update, and update checks select the `stable` channel by default; choose prerelease plugins explicitly with `--channel beta` or `--channel alpha`.
- `ctyun plugin search` supports fuzzy matching and follows the table/JSON output controls.
- `ctyun plugin install` installs only absent plugins; it skips an installed plugin and never upgrades, downgrades, or replaces it through install.
- `ctyun plugin reinstall` operates only on installed plugins and refreshes them from the selected source; reinstall may replace the same version or explicitly move to a lower version from the selected channel.
- `ctyun plugin update` installs only versions with higher SemVer precedence.
- Install, reinstall, update, removal, and core upgrade show progress on stderr in an interactive terminal, then write one summary to stdout; redirected and piped runs emit no progress control sequences.
- `--cols`, `--filter`, and `--sort` accept the column labels shown in the table, while stable column keys remain supported.
- Quote values only when the shell would split them, such as English column labels with spaces.
- Dangerous operations prompt for `y/N` confirmation by default; scripts can use `--yes` or `-y` to skip the prompt.

```sh
ctyun plugin reinstall region --source auto
ctyun plugin reinstall ecs --source auto --channel beta
ctyun plugin reinstall --all --source auto
ctyun plugin update --all --source auto
ctyun plugin update --all --source auto --channel beta
ctyun plugin remove ecs region --yes
```

After installing the matching plugins, common product command shapes look like this:

```sh
ctyun region list
ctyun region list --name 华东1 --cols "Region ID,Region Name,Region Code"
ctyun ecs instance list --cols "Instance ID,Name,Status"
ctyun ecs instance list --name api-test01
ctyun ecs instance show c5a7966a-88e7-362b-6e11-c2d8fbfc07ca
```

Output controls:

```sh
ctyun ecs instance list --output json
ctyun ecs instance list --table compact
ctyun ecs instance list --table plain
ctyun ecs instance list --no-header
ctyun ecs instance list --filter Status=running --sort "-Instance ID"
```

Interactive tables measure Chinese, English, emoji, and other Unicode content by terminal display width and, where possible, wrap at whitespace or common machine-value separators; redirected or piped output retains its natural width. The `bordered`, `compact`, and `plain` styles share the same column-width calculation and wrapping rules.

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

On macOS, Linux, and WSL, use `command -v` to locate the `ctyun` binary on the current `PATH`, then remove it. The default installation path is `$HOME/.local/bin/ctyun`:

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

Values in angle brackets, such as `<name>`, `<plugin-command>`, and paths, are placeholders; replace them with the actual plugin name, command, or path before running the examples.

Development and debugging:

```sh
go run ./cmd/ctyun <plugin-command> --offline
go run ./cmd/ctyun <plugin-command> --fixture
go run ./cmd/ctyun --debug <plugin-command> --offline
```

`--offline` and `--fixture` both enable bundled plugin fixtures and do not call live CTyun APIs. They are long-only product-command options for development builds, must follow the complete product command path, and are not global options. Release builds neither recognize nor expose these development options.

Development builds can use `--bundled` to search, list, install, reinstall, or update plugins from in-tree plugin metadata. Product command execution in development builds also prefers in-tree bundled plugins, so local metadata changes remain visible even when a released plugin with the same name is installed. Like `--fixture`, `--bundled` is for development and test workflows and is omitted from regular help.

```sh
go run ./cmd/ctyun plugin list --available --bundled
go run ./cmd/ctyun plugin search <name> --bundled
go run ./cmd/ctyun plugin install <name> --bundled
go run ./cmd/ctyun plugin reinstall <name> --bundled
go run ./cmd/ctyun plugin update <name> --bundled
```

Testing:

```sh
git ls-files '*.go' | xargs gofmt -w
go vet ./...
go test ./internal/cli -run '^TestGoFilesStayUnderLineLimit$'
go test ./...
go test ./internal/cli -run Completion -v
go test ./tools/plugincheck
go run ./tools/coverage
```

After plugin changes, verify according to the affected area. Lint the changed plugin first, then run the matching offline command. If the change affects generic plugin loading, command parsing, or table rendering, add the related Go tests.

```sh
go run ./cmd/ctyun plugin lint ./plugins/<name>
go run ./cmd/ctyun <plugin-command> --offline

go test ./tools/plugincheck
go test ./internal/cli ./internal/plugin ./internal/output
```

The OpenAPI catalog pipeline is a developer tool. It is not exposed as a user command and is not included in core or plugin release artifacts. It starts from normalized JSON input and stores upstream evidence in `openapi-catalogs/<name>/source.json`:

```sh
go run ./tools/openapi harvest <name> --input path/to/normalized-source.json
go run ./tools/openapi diff <name>
go run ./tools/openapi normalize-labels <name>
go run ./tools/openapi generate <name>
go run ./tools/openapi review <name>
```

For plugins maintained through this pipeline:

- Track the corresponding `source.json` as upstream evidence and the promoted `baseline.json` as the latest accepted snapshot. After upstream evidence changes, drift between `source.json` and the promoted plugin or `baseline.json` is expected until review and promotion; the promoted plugin's source fingerprint and API scope continue to match `baseline.json`.
- Use `product.api_scope` to record the upstream API URI range covered by the plugin; generate, review, and promote flows should not silently include APIs outside that scope.
- For upstream guidance that recommends another API without deprecation or shutdown wording, preserve the target API evidence in `source.json`; if it cannot yet resolve to a tracked, promoted visible command, leave it unresolved and do not generate command-help metadata. Cross-plugin command references remain soft dependencies during plugin loading; once a reference enters promoted repository plugin metadata, release checks must resolve it to the exact non-deprecated target command and reject recommendation cycles.
- Preserve the upstream evidence needed for executable examples in `source.json`: use `request_example` for complete requests and `example` for individual parameter values; after review, record `example_unavailable` explicitly when upstream provides no usable value. Examples that only repeat the command path already shown by Usage, including unresolved path-placeholder forms, are not generated and are rejected by repository release checks; examples should add concrete arguments, meaningful options, structured values, or other behaviour. Review also rejects mechanically assembled English descriptions, examples missing required command options, undeclared options, and values that do not match their parameter type.
- `normalize-labels` applies only conservative shared technical-casing and reviewed-phrase repairs to `source.json`; labels that cannot be repaired reliably remain unchanged and continue to block review.
- Treat `draft/`, `changes.md`, and `review.md` as reproducible local review outputs that are ignored by default; regenerate them with `diff`, `generate`, and `review` when reviewing a product.
- Generated drafts write `source_fingerprint` from `source.json`; existing plugins retain the version, channel, quality, and core compatibility range from their promoted manifests so regeneration cannot downgrade release identity. When the draft passes review and the `generated`/`reviewed`/`curated` quality value truthfully reflects the current curation level, the promote command updates plugin metadata and advances `baseline.json`.
- Keep routine history in git.

```sh
go run ./tools/openapi promote <name>
```

The release packaging tool writes core binary archives, `core-index.json`, `core-index.sig`, installation scripts, plugin archives, `index.json`, and `index.sig`. Development tests use fake HTTP sources to verify signature and download behaviour before public assets exist; real release assets serve the installation, core update, and plugin update flows above.

- The fixed `core` and `plugins` tags are the repository's only two GitHub Release pages and built-artefact roots: `core` stores core installation and update artefacts, while `plugins` stores plugin installation and update artefacts.
- Ordinary version tags continue to be created, but they do not receive separate GitHub Release pages or uploaded artefacts.
- The Gitee `core` Release keeps the same layout. Because Gitee limits attachments per Release, its fixed `plugins` Release stores only `index.json` and `index.sig`; each plugin archive is uploaded to an immutable Gitee Release for its existing `releases/plugins/<name>/<version>` tag. The signed Gitee index uses absolute URLs for those archives, so clients still download only the selected plugin.
- Actual versions and channels are selected by the signed `core-index.json` and `index.json`; version-specific change history is recorded in the canonical root and plugin changelogs.
- When the tool runs against an existing output directory, it preserves existing entries for other channels, replaces the rebuilt core channel or plugin name/channel assets, and signs the merged indexes again; if the same core version is being completed with more platform archives, those platform assets are merged.
- After updating fixed release assets, keep archives referenced by the current signed indexes, index signatures, and core installation scripts, and remove old archives that are no longer referenced; prerelease channel archives should stay when the index still advertises them.

The plugin index and archives at the output root are for GitHub. Use `gitee/index.json` and `gitee/index.sig` to replace the same-named files on Gitee's fixed `plugins` Release. `gitee/releases.json` is a publication manifest, not a user download: it records each archive's plugin, version, channel, target tag, checksum, and expected download URL. Publish every listed immutable Gitee version Release and archive before replacing the fixed `plugins` index.

Core and plugin versions must follow Semantic Versioning 2.0.0. Do not prefix release versions with `v`. Use versions like `0.1.0-alpha.1` with the `alpha`/`beta` channels for pre-releases, and versions like `0.1.0` with the `stable` channel for stable releases. The defaults in `internal/version/version.go` only identify unpackaged development builds, and release packaging overrides the actual version and channel.

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
go run ./tools/release --version 0.4.0 --channel stable --out ./dist/releases --gitee-plugin-download-root "https://gitee.com/ArvinZJC/ctyun-cli/releases/download" --platform "$(go env GOOS)/$(go env GOARCH)"
```

For real releases, GitHub remains the canonical source and CI artifact authority, while Gitee is the synchronised mirror for more reliable access from mainland China. `ctyun` trusts the signing public key and SHA-256 checksums, not the hosting platform itself.

## Related Projects

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli): another unofficial CTyun CLI, written in Python, useful as a reference for users who prefer the Python ecosystem.
