# ctyun-cli

[![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fcore%2F*&label=release)](../../releases)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/ArvinZJC/ctyun-cli)
[![GitHub License](https://img.shields.io/github/license/ArvinZJC/ctyun-cli?label=licence)](./LICENCE)

[简体中文](./README.md) | English

`ctyun` is an unofficial command-line tool written in Go for querying and managing cloud resources through CTyun APIs. It uses product plugins and prioritises the terminal experience, with tables for interactive use and JSON output for scripts. The repository and package are named `ctyun-cli`.

[Install](#installation) · [Configure](#authentication-config-and-language) · [Plugins](#plugins) · [Usage](#using-commands) · [Updates](#core-updates) · [Storage](#storage-authentication-and-file-transfers) · [Contributing](#developer-and-contributor-workflow)

## Highlights

- Readable tables with Chinese and English display-width handling.
- `--output json` for scripts and other tools.
- Product plugins that can be installed and updated independently.
- Waiters for supported resource and task states, with confirmation before dangerous operations.
- Chinese and English (i18n) support for core help, errors, runtime warnings, plugin names, command descriptions, and table labels.

## Before you use it

- **Relationship to the official CLI:** CTyun released the official `ctyun-cli` on 2 July 2026. This project is an independently maintained unofficial implementation.

  The public entry point for CTyun's official `ctyun-cli` is the [official CLI docs](https://www.ctyun.cn/document/11095072). As of now, it does not have a separate official product home page. The official tool uses the `ctyun-cli` command name, while this project uses `ctyun`, so the two binaries do not conflict and can both be present in one shell environment. Both tools use `CTYUN_AK` / `CTYUN_SK` for AK/SK environment variables; if you use them side by side, remember that this credential pair is shared by both tools.

  CTyun's official Go SDK is named `ctyun-go-sdk`, but it has limited product coverage and is not publicly released. Users who need the official SDK can submit a CTyun work order. This project is not an SDK; it is a command-line tool for user workflows.

  This project will keep iterating as an unofficial alternative to the official CLI. We will continue exploring and implementing capabilities expected from a modern cloud CLI, while keeping a better terminal experience, script friendliness, composable output, and maintainable extension paths in mind. Once the official CLI is robust, stable, flexible, capable, and polished enough for the same workflows, we can reassess this project's role and lifecycle.

- Activate the service you want to manage and prepare an AK/SK credential pair with the required permissions.

- This CLI supports customer-side (C-side) APIs for self-operated resource pools (first-class nodes). Business/operations-side (B-side) APIs and joint-operation pools (second-class nodes) are outside its scope.

- Check the [CTyun OpenAPI documentation](https://eop.ctyun.cn/ebp/ctapiDocument/index) for service-specific requirements.

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

The bootstrap scripts download the index and archive from the selected release source (the default sources use HTTPS) and verify the archive against the SHA-256 hash in that index. They do not verify `core-index.sig`; bootstrap therefore trusts the downloaded script and release source. After installation, `ctyun` verifies signed indexes as well as archive hashes for core updates and hosted plugin operations.

The installation scripts support these environment variables:

| Variable                | Purpose                                                                                                                                          |
|-------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------|
| `CTYUN_INSTALL_CHANNEL` | Pin the installation channel to `stable`, `beta`, or `alpha`                                                                                     |
| `CTYUN_INSTALL_SOURCE`  | Pin the installation source to `auto`, `github`, or `gitee`                                                                                      |
| `CTYUN_INSTALL_DIR`     | Override the installation directory; defaults to `$HOME/.local/bin` on macOS, Linux, and WSL, and `%LOCALAPPDATA%\Programs\ctyun-cli` on Windows |

## Authentication, config, and language

Live requests prefer AK/SK from the process environment:

```sh
export CTYUN_AK=...
export CTYUN_SK=...
```

When `CTYUN_AK` or `CTYUN_SK` is missing, `ctyun` falls back to `ak`/`sk` in the active profile, then top-level config. A live command that actually uses config AK/SK writes a warning to stderr; disable it by setting the `CTYUN_WARN_CONFIG_CREDENTIALS=0` environment variable or running `ctyun config set warn_config_credentials false`.

### Configuration and profiles

The config file is selected by `--config`, then `CTYUN_CONFIG`, then `~/.ctyun/config.json`. `--profile` overrides `active_profile`. Other settings use command-line options first, then environment variables, the active profile, and supported top-level defaults.

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

Security recommendations:

- Prefer environment variables for AK/SK; if you store them in config, keep the file out of repositories and restrict its permissions.
- Do not store AK/SK in scripts, shell history, or logs.
- Use least-privilege IAM user AK/SK for `ctyun` and rotate them regularly.
- Avoid exposing environment variables on shared machines or in CI logs.
- When using `--debug`, inspect logs again before sharing them.

Config files can hold resource pool, language, timeout, advanced API endpoint overrides for testing or private environments, warning preferences, and fallback values for `CTYUN_AK`/`CTYUN_SK`. Global keys are `active_profile`, `ak`, `sk`, `warn_config_credentials`, and `warn_deprecated`. Profile keys are `region`, `language`, `endpoint_url`, `timeout_seconds`, `ak`, `sk`, `warn_config_credentials`, and `warn_deprecated`; command help lists their accepted value shapes and fixed defaults.

```json
{
  "profiles": {
    "prod": {
      "region": "81f7728662dd11ec810800155d307d5b"
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
```

`ctyun config show` displays stored JSON and masks saved AK/SK values like `aa*****dd`; unset values are omitted. `ctyun config explain` instead reports effective base settings and the source that won for each value. Sensitive rows report only whether a value is configured and never reveal, mask, fingerprint, or otherwise derive AK/SK.

Use `ctyun doctor local` to check configuration, credentials, resource pool settings, and installed plugins without network requests or local changes. It reports all independent findings and exits with status 1 if any check fails; warnings and skipped checks exit with status 0. Use `ctyun doctor network` for online source and API endpoint diagnostics.

`ctyun config reset` prompts for confirmation, then creates a backup before deleting the current config file. Scripts can use `--yes` or `-y` to skip the prompt.

### Language

Supported languages are `zh-CN`, `en-US`, and `en-GB`. Language resolution is `--lang`, then `CTYUN_LANGUAGE`, then profile `language`, then the OS locale. If nothing matches, `zh-CN` is used.

## Plugins

A fresh `ctyun` installation includes only core commands; product plugins are not preinstalled. Product commands come from plugin bundles. After setting up authentication, config, and language preferences, install the plugins you need:

```sh
ctyun plugin search ecs --source auto
ctyun plugin list --available --source auto
ctyun plugin list --available --cols Plugin,Quality,Status --filter Status=available --source auto
ctyun plugin install region --source auto
ctyun plugin install ecs --source auto --channel beta
ctyun plugin install --all --source auto
ctyun plugin list
```

<details>
<summary>Plugin table</summary>

| Name                                      | Plugin                   | Product                  | Version                                                                                                                                                      | Channel  | Quality     | Commands | Operations |
|-------------------------------------------|--------------------------|--------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|-------------|---------:|-----------:|
| Application Cloud Server                  | `acs`                    | `acs`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Facs%2F*&label=release)](../../releases)                    | `beta`   | `generated` |       15 |         15 |
| Auto Scaling                              | `as`                     | `as`                     | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fas%2F*&label=release)](../../releases)                     | `beta`   | `generated` |       63 |         63 |
| Cloud Backup and Recovery                 | `cbr`                    | `cbr`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcbr%2F*&label=release)](../../releases)                    | `beta`   | `generated` |       22 |         22 |
| Cloud Disaster Recovery                   | `cdr`                    | `cdr`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcdr%2F*&label=release)](../../releases)                    | `beta`   | `generated` |       31 |         31 |
| Cloud Function                            | `cf`                     | `cf`                     | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcf%2F*&label=release)](../../releases)                     | `beta`   | `generated` |       62 |         62 |
| Cloud Assistant                           | `cloud-assistant`        | `cloud-assistant`        | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcloud-assistant%2F*&label=release)](../../releases)        | `beta`   | `generated` |       11 |         11 |
| Common                                    | `common`                 | `common`                 | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcommon%2F*&label=release)](../../releases)                 | `stable` | `curated`   |        1 |          1 |
| Dedicated Physical Server                 | `dps`                    | `dps`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fdps%2F*&label=release)](../../releases)                    | `beta`   | `generated` |       59 |         59 |
| CTyun Cloud Computer (Enterprise Edition) | `ecpc`                   | `ecpc`                   | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fecpc%2F*&label=release)](../../releases)                   | `beta`   | `generated` |      303 |        303 |
| Elastic Cloud Server                      | `ecs`                    | `ecs`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fecs%2F*&label=release)](../../releases)                    | `beta`   | `generated` |      209 |        209 |
| Elastic High Performance Computing        | `ehpc`                   | `ehpc`                   | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fehpc%2F*&label=release)](../../releases)                   | `beta`   | `generated` |       24 |         24 |
| Elastic Volume Service                    | `evs`                    | `evs`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fevs%2F*&label=release)](../../releases)                    | `beta`   | `generated` |       36 |         36 |
| High Performance File Storage             | `hpfs`                   | `hpfs`                   | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fhpfs%2F*&label=release)](../../releases)                   | `beta`   | `generated` |       40 |         40 |
| Image Management Service                  | `ims`                    | `ims`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fims%2F*&label=release)](../../releases)                    | `beta`   | `generated` |       27 |         27 |
| Job                                       | `job`                    | `job`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fjob%2F*&label=release)](../../releases)                    | `stable` | `curated`   |        1 |          1 |
| OceanFS                                   | `oceanfs`                | `oceanfs`                | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Foceanfs%2F*&label=release)](../../releases)                | `beta`   | `generated` |       34 |         34 |
| Classic Object Storage Type I             | `classic-object-storage` | `classic-object-storage` | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fclassic-object-storage%2F*&label=release)](../../releases) | `beta`   | `generated` |      219 |        219 |
| Media Storage                             | `media-storage`          | `media-storage`          | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fmedia-storage%2F*&label=release)](../../releases)          | `beta`   | `generated` |      124 |        124 |
| Order                                     | `order`                  | `order`                  | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Forder%2F*&label=release)](../../releases)                  | `stable` | `curated`   |        7 |          7 |
| Relational Database for MySQL             | `rds-mysql`              | `rds-mysql`              | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Frds-mysql%2F*&label=release)](../../releases)              | `beta`   | `generated` |      224 |        224 |
| Relational Database for PostgreSQL        | `rds-postgresql`         | `rds-postgresql`         | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Frds-postgresql%2F*&label=release)](../../releases)         | `beta`   | `generated` |      153 |        153 |
| SQL Server                                | `rds-sqlserver`          | `rds-sqlserver`          | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Frds-sqlserver%2F*&label=release)](../../releases)          | `beta`   | `generated` |      114 |        114 |
| Region                                    | `region`                 | `region`                 | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fregion%2F*&label=release)](../../releases)                 | `stable` | `curated`   |        7 |          7 |
| Scalable File Service                     | `sfs`                    | `sfs`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fsfs%2F*&label=release)](../../releases)                    | `beta`   | `generated` |       56 |         56 |
| Volume Backup Service                     | `vbs`                    | `vbs`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fvbs%2F*&label=release)](../../releases)                    | `beta`   | `generated` |       37 |         37 |
| Zettabyte Object Storage                  | `zos`                    | `zos`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fzos%2F*&label=release)](../../releases)                    | `beta`   | `generated` |      106 |        106 |

The quality field describes plugin metadata maturity: `generated` is a tool-generated draft, `reviewed` has passed a project review, and `curated` is kept as a maintained reference set.

</details>

- RDS: six PostgreSQL download/export APIs are not yet supported; see the [coverage inventory](openapi-catalogs/rds-postgresql/coverage.json). The retired MySQL cross-region backup destination query remains listed with a deprecation warning but is unavailable online.
- Storage: `media-storage` and `classic-object-storage` commands under `native` use separate storage credentials. See [Storage authentication and file transfers](#storage-authentication-and-file-transfers). The media-storage OpenAPI gateway applies to Tibet resource pool 1.
- Cloud Assistant: `ctyun cloud-assistant command run` is temporarily unavailable because the upstream API is suspended. The command is retained; this is not permanent deprecation.

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

```sh
ctyun plugin reinstall region --source auto
ctyun plugin reinstall ecs --source auto --channel beta
ctyun plugin reinstall --all --source auto
ctyun plugin update --all --source auto
ctyun plugin update --all --source auto --channel beta
ctyun plugin remove ecs region --yes
```

## Using commands

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

### Product commands

Dangerous operations prompt for `y/N` confirmation by default; scripts can use `--yes` or `-y` to skip the prompt.

After installing the matching plugins, common product command shapes look like this:

```sh
ctyun region list
ctyun region list --name 华东1 --cols "Region ID,Region Name,Region Code"
ctyun ecs instance list --cols "Instance ID,Name,Status"
ctyun ecs instance list --name api-test01
ctyun ecs instance show c5a7966a-88e7-362b-6e11-c2d8fbfc07ca
```

### Output and filtering

- `--cols`, `--filter`, and `--sort` accept the column labels shown in the table, while stable column keys remain supported.
- Quote values only when the shell would split them, such as English column labels with spaces.

```sh
ctyun ecs instance list --output json
ctyun ecs instance list --table compact
ctyun ecs instance list --table plain
ctyun ecs instance list --no-header
ctyun ecs instance list --filter Status=running --sort "-Instance ID"
```

Interactive tables measure Chinese, English, emoji, and other Unicode content by terminal display width and, where possible, wrap at whitespace or common machine-value separators; redirected or piped output retains its natural width. The `bordered`, `compact`, and `plain` styles share the same column-width calculation and wrapping rules.

### Waiting for resource or task state

Waiters require core `>=0.5.0 <1.0.0`. Use a retrieval command with `--wait <waiter>` to poll until its documented target state is observed. Command help lists applicable waiters, and shell completion offers the same choices. Waiters are checked before a request; incompatible commands and operations without safe polling metadata are rejected. Polling repeats the retrieval command, so use the resource or task identifier returned by the earlier mutation.

```sh
ctyun ecs instance show {instance_id} --wait ecs.instance.running
ctyun vbs backup show --backup-id <backup_id> --wait vbs.backup.available
ctyun hpfs dataflow-task show --task-id <task_id> --wait hpfs.dataflow-task.completed
ctyun evs snapshot list --snapshot-id <snapshot_id> --wait evs.snapshot.available
ctyun ims image show {image_id} --wait ims.image.active
```

For waits on a collection, provide the resource identifier required by command help. Each response must contain an exact match: a missing resource stays pending, and duplicate matches are rejected.

Each waiter defines `max_attempts` (including the first request) and `interval_seconds`; these are separate from the HTTP request timeout. Null and unknown values stay pending until the limit. Failure and timeout are printed as final waiter states; they currently do not change the command exit status. JSON output remains the initial response on stdout, with the waiter status on stderr.

## Core updates

Use `ctyun update` or `ctyun upgrade` to check for and install core updates. Choose `auto`, `github`, or `gitee` with `--source`; `auto` tries GitHub first and falls back to Gitee. Updates verify index signatures and archive SHA-256 hashes. Use `--channel` to select `stable`, `beta`, or `alpha`.

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

## Storage authentication and file transfers

### Native storage credentials

The `media-storage` and `classic-object-storage` plugins require core `>=0.5.0 <1.0.0`. Their `native` commands use `CTYUN_STORAGE_ENDPOINT` (an HTTPS service endpoint without a bucket or path), `CTYUN_STORAGE_AK`, and `CTYUN_STORAGE_SK`. Set `CTYUN_STORAGE_REGION` to the signing region in the service documentation, not an OpenAPI resource-pool ID. `CTYUN_STORAGE_SIGNATURE_VERSION` defaults to `v4`; `v2` does not require a signing region. Use `CTYUN_STORAGE_SECURITY_TOKEN` for temporary credentials in header-signed requests. EOP profile endpoints and credentials are not reused. Media uses path-style bucket addressing; classic uses virtual-host addressing.

For classic storage, `native statistics` uses signing service `s3` and a region such as `cn-mg`; `native tracking` uses `cloudtrail`, and `native iam` uses `sts`, both with a region such as `cn`. Select the endpoint and signing region from the corresponding service documentation. Tracking and IAM require V4. IAM `--tags` accepts a JSON array of `Key`/`Value` objects, and `--tag-keys` accepts a JSON string array. Supply form values unencoded; the CLI URL-encodes them.

### Policy-based uploads

For `ctyun media-storage object post {bucket_name}`, provide `--key` and `--file`. Storage authentication also requires `--storage-access-key` and a base64-encoded JSON `--policy`; supply `--signature` or set `CTYUN_STORAGE_SK` for local V2 signing. Gateway authentication uses `CTYUN_AK`/`CTYUN_SK`. Native `object post` uses the same V2 policy inputs and only `CTYUN_STORAGE_ENDPOINT` for routing; `CTYUN_STORAGE_SIGNATURE_VERSION` does not change its policy algorithm. Pass temporary storage credentials with `--security-token`. The OpenAPI gateway uses `success-action-status` and `success-action-redirect`; native forms use `success_action_status` and `success_action_redirect`. Successful redirects are reported without being followed.

### File input and output

Use `--file` or `--document-file` only on commands that declare file inputs; ordinary values beginning with `@` are not read as files. Native uploads offer `--content-type` and, where supported, `--metadata` as a JSON map of metadata names to string values. Uploads require temporary disk space. Structured responses and XML documents are limited to 16 MiB. Command help lists available output formats: `--output raw` writes exact response bytes and cannot be combined with table controls or waiters; downloads also offer `--output-file` and `--overwrite`.

### Pagination and waiters

Media native pagination is explicit: read continuation markers with `--output raw` or `--output json` and pass them to the next request. The object availability waiter recognises valid restored objects and non-archived objects. Tracking waiters check whether logging is enabled or disabled, not whether log delivery is healthy.

## Developer and contributor workflow

Development and builds require Go 1.26.0 or later to meet the minimum requirement of the current dependencies.

The project follows the [upstream Go support window](https://go.dev/doc/devel/release#policy) by default, supporting the two latest Go release families and normally using the older family's `.0` release as the minimum. Language features, standard-library APIs, dependencies, or necessary compiler/runtime fixes may require a higher minimum; outdated dependencies should not be retained indefinitely to support end-of-life Go versions. The `go` directive in `go.mod` records the minimum requirement; development and release builds use the latest patch of the latest stable Go release. A local toolchain upgrade alone does not require raising the minimum or adding a `toolchain` directive.

If the default Go build cache is not writable, for example in a sandbox, use a repo-local cache first:

```sh
export GOCACHE="$PWD/.cache/go-build"
```

Positional argument placeholders use braces, such as `{instance_id}`; option values and developer shorthand such as `<name>` and `<plugin-command>` use angle brackets. Replace placeholders with actual values, commands, or paths before running the examples.

### Build and test

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
git ls-files -co --exclude-standard '*.go' | sort -u | xargs gofmt -w
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

### Maintaining plugin catalogs

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
- Assess lifecycle waiters during each plugin review. Catalog `waiters` maps waiter IDs to `commands` (exact command IDs), a state `path` (JSON) or `xml_path` (an absolute namespace-aware XML path selecting exactly one scalar element, without a collection selector), scalar `success`/`failure`, optional additional `success_values`/`failure_values`, positive `max_attempts`/`interval_seconds`, and an `evidence` note identifying the documented state semantics. Empty scalar failure means no documented failure condition. Explicit bindings must reference non-dangerous retryable retrieval operations. Review rejects draft waiter drift; promotion preserves the definitions and advances the baseline. Add offline bundle checks for response shape, terminal outcomes, and command-specific help/completion. For collection responses, add `selector` with the collection `path`, row identity `key`, and `value` referencing an existing `$arg.<name>` or string/integer `$param.<name>`; the state path is relative to the matched row. Check every captured row, preserve exact numeric identities, and keep null or unknown states pending. Record gaps when state evidence or a unique identity input is missing.
- When a published API has an explicit response contract but no usable successful response example, record the reason in `fixture_unavailable`. For downloads with a documented success media type, declare `media_type` on the response variant so HTTP-200 error envelopes cannot be saved as files. Retain the command and response validation without generating a success fixture, and keep the original response evidence in the catalog. Such commands cannot run with `--offline` or `--fixture` and cannot supply response evidence for waiters.
- Use `product.api_scope` to record the upstream API URI range covered by the plugin; generate, review, and promote flows should not silently include APIs outside that scope.
- For upstream guidance that recommends another API without deprecation or shutdown wording, preserve the target API evidence in `source.json`; if it cannot yet resolve to a tracked, promoted visible command, leave it unresolved and do not generate command-help metadata. Cross-plugin command references remain soft dependencies during plugin loading; once a reference enters promoted repository plugin metadata, release checks must resolve it to the exact non-deprecated target command and reject recommendation cycles.
- Preserve the upstream evidence needed for executable examples in `source.json`: use `request_example` for complete requests and `example` for individual parameter values; after review, record `example_unavailable` explicitly when upstream provides no usable value. Examples that only repeat the command path already shown by Usage, including unresolved path-placeholder forms, are not generated and are rejected by repository release checks; examples should add concrete arguments, meaningful options, structured values, or other behaviour. Review also rejects mechanically assembled English descriptions, examples missing required command options, undeclared options, and values that do not match their parameter type.
- `normalize-labels` applies only conservative shared technical-casing and reviewed-phrase repairs to `source.json`; labels that cannot be repaired reliably remain unchanged and continue to block review.
- Treat `draft/`, `changes.md`, and `review.md` as reproducible local review outputs that are ignored by default; regenerate them with `diff`, `generate`, and `review` when reviewing a product.
- Generated drafts write `source_fingerprint` from `source.json`; existing plugins retain the version, channel, quality, and core compatibility range from their promoted manifests, raising the minimum to 0.5.0 when catalog waiters require it so regeneration cannot downgrade release identity. When the draft passes review and the `generated`/`reviewed`/`curated` quality value truthfully reflects the current curation level, the promote command updates plugin metadata and advances `baseline.json`.
- Keep routine history in git.

```sh
go run ./tools/openapi promote <name>
```

### Packaging and releases

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
go run ./tools/release --version 0.5.0 --channel stable --out ./dist/releases --gitee-plugin-download-root "https://gitee.com/ArvinZJC/ctyun-cli/releases/download" --platform "$(go env GOOS)/$(go env GOARCH)"
```

For real releases, GitHub remains the canonical source and CI artifact authority, while Gitee is the synchronised mirror for more reliable access from mainland China. The installed `ctyun` verifies hosted core and plugin indexes using the signing public key and checks archive SHA-256 hashes; the bootstrap scripts use the installation trust model described above.

## Related projects

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli): another unofficial CTyun CLI, written in Python, useful as a reference for users who prefer the Python ecosystem.
