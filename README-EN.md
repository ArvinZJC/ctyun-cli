# ctyun-cli

[![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fcore%2Fcore-index.json&query=%24.versions.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/core)
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

- This CLI supports customer-side (C-side) APIs for self-operated resource pools (first-class nodes). Business/operations-side (B-side) APIs and joint-operation pools (second-class nodes) are outside its scope. This node classification is distinct from product-specific Type I/II resource-pool versions, such as those in Redis documentation; the numeral alone does not determine whether an API is in scope.

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

| Name                                                                | Plugin                       | Product                      | Version                                                                                                                                                                                                                                                                                                               | Channel  | Quality     | Commands | Operations |
|---------------------------------------------------------------------|------------------------------|------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|-------------|---------:|-----------:|
| Application Cloud Server                                            | `acs`                        | `acs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27acs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       15 |         15 |
| Agent Engine                                                        | `agent-engine`               | `agent-engine`               | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27agent-engine%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)               | `beta`   | `generated` |       11 |         11 |
| AI Capability Platform                                              | `ai-capability`              | `ai-capability`              | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ai-capability%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)              | `beta`   | `generated` |       41 |         41 |
| AI Security                                                         | `ai-security`                | `ai-security`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ai-security%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |        5 |          5 |
| API Gateway                                                         | `api-gateway`                | `api-gateway`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27api-gateway%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |      161 |        161 |
| Application Performance Monitoring                                  | `apm`                        | `apm`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27apm%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       63 |         63 |
| Application Acceleration                                            | `application-acceleration`   | `application-acceleration`   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27application-acceleration%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)   | `beta`   | `generated` |       12 |         12 |
| Application High Availability                                       | `application-ha`             | `application-ha`             | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27application-ha%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)             | `beta`   | `generated` |       43 |         43 |
| Auto Scaling                                                        | `as`                         | `as`                         | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27as%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                         | `beta`   | `generated` |       63 |         63 |
| Billing                                                             | `billing`                    | `billing`                    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27billing%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                    | `beta`   | `generated` |       16 |         16 |
| Cloud Application Engine                                            | `cae`                        | `cae`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cae%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       44 |         44 |
| Cloud Backup and Recovery                                           | `cbr`                        | `cbr`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cbr%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       22 |         22 |
| Cloud Container Engine                                              | `cce`                        | `cce`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cce%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |      306 |        306 |
| CCE One                                                             | `cce-one`                    | `cce-one`                    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cce-one%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                    | `beta`   | `generated` |       41 |         41 |
| Cloud Dedicated Access                                              | `cda`                        | `cda`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cda%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       61 |         61 |
| Content Delivery Network Acceleration                               | `cdn`                        | `cdn`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cdn%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       92 |         92 |
| Cloud Disaster Recovery                                             | `cdr`                        | `cdr`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cdr%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       31 |         31 |
| Certificate Management Service                                      | `certificate`                | `certificate`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27certificate%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |       14 |         14 |
| Cloud Function                                                      | `cf`                         | `cf`                         | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cf%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                         | `beta`   | `generated` |       62 |         62 |
| Classic Object Storage Type I                                       | `classic-object-storage`     | `classic-object-storage`     | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27classic-object-storage%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)     | `beta`   | `generated` |      219 |        219 |
| ClickHouse                                                          | `clickhouse`                 | `clickhouse`                 | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27clickhouse%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                 | `beta`   | `generated` |       54 |         54 |
| Cloud Assistant                                                     | `cloud-assistant`            | `cloud-assistant`            | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cloud-assistant%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)            | `beta`   | `generated` |       11 |         11 |
| Cloud Audit                                                         | `cloud-audit`                | `cloud-audit`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cloud-audit%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |       13 |         13 |
| Cloud Native API Gateway                                            | `cloud-native-api-gateway`   | `cloud-native-api-gateway`   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cloud-native-api-gateway%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)   | `beta`   | `generated` |      186 |        186 |
| Cloud Search Service                                                | `cloud-search`               | `cloud-search`               | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cloud-search%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)               | `beta`   | `generated` |        7 |          7 |
| Cloud Monitoring Service                                            | `cm`                         | `cm`                         | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cm%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                         | `beta`   | `generated` |      169 |        169 |
| Common                                                              | `common`                     | `common`                     | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27common%27%5D.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                     | `stable` | `curated`   |        1 |          1 |
| Container Registry Service                                          | `crs`                        | `crs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27crs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |      269 |        269 |
| Dedicated Cloud (Dedicated Computing)                               | `dec`                        | `dec`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27dec%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |        9 |          9 |
| Data Management Service                                             | `dms`                        | `dms`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27dms%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       78 |         78 |
| Dedicated Physical Server                                           | `dps`                        | `dps`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27dps%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       60 |         60 |
| Distributed Relational Database                                     | `drds`                       | `drds`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27drds%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |      125 |        125 |
| Data Transmission Service                                           | `dts`                        | `dts`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27dts%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       72 |         72 |
| Express Connect (Standard Edition)                                  | `ec`                         | `ec`                         | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ec%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                         | `beta`   | `generated` |       51 |         51 |
| Elastic Container Instances                                         | `eci`                        | `eci`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27eci%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       37 |         37 |
| CTyun Cloud Computer (Enterprise Edition)                           | `ecpc`                       | `ecpc`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ecpc%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |      303 |        303 |
| Elastic Cloud Server                                                | `ecs`                        | `ecs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ecs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |      209 |        209 |
| Intelligent Edge Cloud                                              | `ecx`                        | `ecx`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ecx%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |      252 |        252 |
| DDoS Protection (Edge Cloud Edition)                                | `edge-ddos`                  | `edge-ddos`                  | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27edge-ddos%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                  | `beta`   | `generated` |       39 |         39 |
| Edge Security Acceleration Platform                                 | `edge-security-acceleration` | `edge-security-acceleration` | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27edge-security-acceleration%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins) | `beta`   | `generated` |       79 |         79 |
| Web Application Firewall (Edge Cloud Edition)                       | `edge-waf`                   | `edge-waf`                   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27edge-waf%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                   | `beta`   | `generated` |       68 |         68 |
| Elastic High Performance Computing                                  | `ehpc`                       | `ehpc`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ehpc%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |       40 |         40 |
| Elastic IP                                                          | `eip`                        | `eip`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27eip%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       57 |         57 |
| Elastic Load Balancing                                              | `elb`                        | `elb`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27elb%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |      114 |        114 |
| MapReduce                                                           | `emr`                        | `emr`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27emr%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       13 |         13 |
| Elastic Volume Service                                              | `evs`                        | `evs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27evs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       36 |         36 |
| Global Traffic Management                                           | `gtm`                        | `gtm`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27gtm%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       21 |         21 |
| High Performance File Storage                                       | `hpfs`                       | `hpfs`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27hpfs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |       40 |         40 |
| Distributed Hybrid Transactional and Analytical Processing Database | `htap`                       | `htap`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27htap%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |       47 |         47 |
| Huiju Integrated Intelligent Computing Service Platform             | `huiju`                      | `huiju`                      | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27huiju%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `beta`   | `generated` |       86 |         86 |
| Identity and Access Management                                      | `iam`                        | `iam`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27iam%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       87 |         87 |
| Image Management Service                                            | `ims`                        | `ims`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ims%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       27 |         27 |
| Influx                                                              | `influxdb`                   | `influxdb`                   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27influxdb%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                   | `beta`   | `generated` |       48 |         48 |
| Job                                                                 | `job`                        | `job`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27job%27%5D.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `stable` | `curated`   |        1 |          1 |
| Kafka                                                               | `kafka`                      | `kafka`                      | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27kafka%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `beta`   | `generated` |      124 |        124 |
| Key Management                                                      | `kms`                        | `kms`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27kms%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       82 |         82 |
| Live Streaming                                                      | `live-streaming`             | `live-streaming`             | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27live-streaming%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)             | `beta`   | `generated` |       90 |         90 |
| Log Service                                                         | `lts`                        | `lts`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27lts%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |      171 |        171 |
| Media Storage                                                       | `media-storage`              | `media-storage`              | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27media-storage%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)              | `beta`   | `generated` |      124 |        124 |
| Model Training and Inference                                        | `model-training`             | `model-training`             | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27model-training%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)             | `beta`   | `generated` |       67 |         67 |
| Document Database Service                                           | `mongodb`                    | `mongodb`                    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27mongodb%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                    | `beta`   | `generated` |      180 |        180 |
| Distributed Message Queuing Telemetry Transport                     | `mqtt`                       | `mqtt`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27mqtt%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |       61 |         61 |
| Microservice Cloud Application Platform                             | `msap`                       | `msap`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27msap%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |      157 |        157 |
| Microservice Engine                                                 | `mse`                        | `mse`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27mse%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |      243 |        243 |
| NAT Gateway                                                         | `nat`                        | `nat`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27nat%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       22 |         22 |
| Cloud Firewall (Native Edition)                                     | `native-firewall`            | `native-firewall`            | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27native-firewall%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)            | `beta`   | `generated` |      104 |        104 |
| OceanFS                                                             | `oceanfs`                    | `oceanfs`                    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27oceanfs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                    | `beta`   | `generated` |       34 |         34 |
| Order                                                               | `order`                      | `order`                      | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27order%27%5D.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `stable` | `curated`   |        7 |          7 |
| Private DNS                                                         | `private-dns`                | `private-dns`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27private-dns%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |       40 |         40 |
| Private NAT Gateway                                                 | `private-nat`                | `private-nat`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27private-nat%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |       21 |         21 |
| RabbitMQ                                                            | `rabbitmq`                   | `rabbitmq`                   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27rabbitmq%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                   | `beta`   | `generated` |       39 |         39 |
| Relational Database for MySQL                                       | `rds-mysql`                  | `rds-mysql`                  | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27rds-mysql%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                  | `beta`   | `generated` |      224 |        224 |
| Relational Database for PostgreSQL                                  | `rds-postgresql`             | `rds-postgresql`             | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27rds-postgresql%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)             | `beta`   | `generated` |      156 |        156 |
| SQL Server                                                          | `rds-sqlserver`              | `rds-sqlserver`              | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27rds-sqlserver%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)              | `beta`   | `generated` |      116 |        116 |
| Distributed Cache Service for Redis                                 | `redis`                      | `redis`                      | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27redis%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `beta`   | `generated` |      207 |        207 |
| Region                                                              | `region`                     | `region`                     | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27region%27%5D.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                     | `stable` | `curated`   |        7 |          7 |
| Research Assistant                                                  | `research-assistant`         | `research-assistant`         | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27research-assistant%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)         | `beta`   | `generated` |       47 |         47 |
| RocketMQ                                                            | `rocketmq`                   | `rocketmq`                   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27rocketmq%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                   | `beta`   | `generated` |       46 |         46 |
| Resource Orchestration Service                                      | `ros`                        | `ros`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ros%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       40 |         40 |
| SD-WAN                                                              | `sdwan`                      | `sdwan`                      | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27sdwan%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `beta`   | `generated` |      138 |        138 |
| Server Security (Native Edition)                                    | `server-security`            | `server-security`            | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27server-security%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)            | `beta`   | `generated` |      154 |        154 |
| Application Service Mesh                                            | `service-mesh`               | `service-mesh`               | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27service-mesh%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)               | `beta`   | `generated` |       23 |         23 |
| Scalable File Service                                               | `sfs`                        | `sfs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27sfs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       56 |         56 |
| Smart Domain Name System                                            | `smart-dns`                  | `smart-dns`                  | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27smart-dns%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                  | `beta`   | `generated` |       19 |         19 |
| Cloud Messaging - SMS                                               | `sms`                        | `sms`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27sms%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       24 |         24 |
| Xingchen TokenHub Operations Platform                               | `tokenhub`                   | `tokenhub`                   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27tokenhub%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                   | `beta`   | `generated` |       19 |         19 |
| Shared Traffic Packages                                             | `traffic-package`            | `traffic-package`            | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27traffic-package%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)            | `beta`   | `generated` |        6 |          6 |
| Volume Backup Service                                               | `vbs`                        | `vbs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27vbs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       37 |         37 |
| Intelligent Visual Service                                          | `visual-service`             | `visual-service`             | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27visual-service%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)             | `beta`   | `generated` |       46 |         46 |
| Video on Demand                                                     | `vod`                        | `vod`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27vod%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       38 |         38 |
| Virtual Private Cloud                                               | `vpc`                        | `vpc`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27vpc%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |      207 |        207 |
| VPC Endpoints                                                       | `vpce`                       | `vpce`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27vpce%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |       47 |         47 |
| Web Application Firewall (Native Edition)                           | `waf`                        | `waf`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27waf%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |       94 |         94 |
| Whole Site Acceleration                                             | `whole-site-acceleration`    | `whole-site-acceleration`    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27whole-site-acceleration%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)    | `beta`   | `generated` |       96 |         96 |
| Yunxiao Intelligent Computing Platform                              | `yunxiao`                    | `yunxiao`                    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27yunxiao%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                    | `beta`   | `generated` |       88 |         88 |
| Zettabyte Object Storage                                            | `zos`                        | `zos`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27zos%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |      106 |        106 |

The quality field describes plugin metadata maturity: `generated` is a tool-generated draft, `reviewed` has passed a project review, and `curated` is kept as a maintained reference set.

</details>

The CLI provides 96 plugins and 7,638 commands for customer-side APIs in self-operated resource pools. Some products or APIs remain unsupported because their public endpoints or request and response contracts cannot be verified. Joint-operation pools and other out-of-scope APIs are excluded. Coverage is based on verified official documentation and does not imply that every API has been tested against a live cloud environment.

The milestone targets core `0.5.1`, which supports all plugins listed above; 45 plugins require at least this version. Plugin versions and release channels remain independent of the core version.

When using the following plugins, note their scope, configuration requirements, and limitations:

- Gateways: `api-gateway` covers the resource and order proxy catalog for self-operated customer-side usage. `cloud-native-api-gateway` manages standalone cloud-native API gateway instances; gateway commands in `mse` belong to Microservice Engine. These use different endpoints and API contracts.

- Networking: `nat` manages public NAT gateways; `private-nat` manages private NAT gateways. The legacy EIP network query is not yet supported.
- RDS: six PostgreSQL download/export APIs are not yet supported. The retired MySQL cross-region backup destination query remains listed with a deprecation warning but is unavailable online.
- Redis: `redis` is based on the API catalog officially labeled “2023-10-18 II类资源池” (`vid=270`). This Redis catalog label does not mean the joint-operation pools (second-class nodes) excluded by this project. Historical APIs are under `ctyun redis legacy`; it requires core `>=0.5.1 <1.0.0`. All commands include offline response fixtures; waiting for completion is not yet supported for some lifecycle operations.
- MongoDB: older APIs are under `ctyun mongodb v1`; the plugin requires core `>=0.5.1 <1.0.0`. Three export APIs remain unavailable because their binary response contracts are unclear. All supported commands include offline response fixtures.
- Storage: `media-storage` and `classic-object-storage` commands under `native` use separate storage credentials. See [Storage authentication and file transfers](#storage-authentication-and-file-transfers). The media-storage OpenAPI gateway applies to Tibet resource pool 1.
- Cloud Assistant: the API used by `ctyun cloud-assistant command run` is available, but its public API documentation is currently unavailable.

Plugin management commands share these behaviours:

- `ctyun plugin search`, `ctyun plugin list --available`, `ctyun plugin install`, `ctyun plugin reinstall`, and `ctyun plugin update` support `--source` and `--channel`.
- `ctyun plugin list --available` shows hosted plugins with local installation status.
- `ctyun plugin list --available` and `ctyun plugin search` inspect the `stable` channel by default, and can use `--channel all` to inspect every registry channel.
- Install, reinstall, update, and update checks select the `stable` channel by default; choose prerelease plugins explicitly with `--channel beta` or `--channel alpha`.
- `ctyun plugin search` supports fuzzy matching and follows the table/JSON output controls.
- `ctyun plugin install` installs only absent plugins; it skips an installed plugin and never upgrades, downgrades, or replaces it through installation.
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
go test -timeout=30m ./...
go test ./internal/cli -run Completion -v
go test -timeout=30m ./tools/plugincheck
go run ./tools/coverage
```

The coverage gate requires 100% statement coverage of the unfiltered profile at `.cache/coverage/raw.out` for the current Go platform. It checks execution counts directly, with no file or line exclusions and no rounding allowance. Tests include the command entry points, maintenance tools, archive helpers, and filesystem failure paths.

After plugin changes, verify according to the affected area. Lint the changed plugin first, then run the matching offline command. If the change affects generic plugin loading, command parsing, or table rendering, add the related Go tests.

```sh
go run ./cmd/ctyun plugin lint ./plugins/<name>
go run ./cmd/ctyun <plugin-command> --offline

go test -timeout=30m ./tools/plugincheck
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

The following machine-readable records support plugin maintenance by developers and AI. The [product coverage inventory](openapi-catalogs/product-coverage.json) compares promoted commands with the official product catalog, including cross-product references, reviewed equivalent APIs, excluded APIs, and confirmed documentation blockers. Equivalent coverage uses the target product endpoint and response contract; it does not claim support for the older catalog’s response shape. The [official API review](openapi-catalogs/official-review.json) records the latest full-plugin check, evidence-backed updates, retained documentation gaps, and endpoint blockers.

Product-level records detail API gaps, source corrections or waiter limitations for [EIP](openapi-catalogs/eip/coverage.json), [PostgreSQL](openapi-catalogs/rds-postgresql/coverage.json), [Redis](openapi-catalogs/redis/coverage.json) and [MongoDB](openapi-catalogs/mongodb/coverage.json).

Update plugins in four stages: prepare the evidence, review the generated draft, check lifecycle waiters, then promote the accepted result.

#### 1. Update the source evidence

Keep these files under `openapi-catalogs/<name>/`:

| File                                | Purpose                                                       | Git policy                     |
|-------------------------------------|---------------------------------------------------------------|--------------------------------|
| `source.json`                       | Current upstream evidence, API scope and metadata to generate | Track                          |
| `baseline.json`                     | Last accepted snapshot used to validate the promoted plugin   | Track; updated by promotion    |
| `draft/`, `changes.md`, `review.md` | Generated bundle, change report and review results            | Ignore; regenerate when needed |

A newer `source.json` may differ from `baseline.json` while changes await review. Until promotion, validate the existing plugin's source fingerprint and API scope against `baseline.json`. Keep routine history in git.

When updating `source.json`:

- Declare the supported API URI ranges in `product.api_scope`. Generation, review and promotion must stay within those ranges.
- Capture complete request examples in `request_example` and individual parameter values in `example`. If upstream provides no usable example, confirm that gap during review and record `example_unavailable`.
- Preserve the evidence for recommended alternative APIs. A recommendation without deprecation or shutdown wording must not become a deprecation warning.

#### 2. Generate and review the draft

Use `diff`, `generate` and `review` to rebuild the local outputs. `normalize-labels` can repair shared technical casing and reviewed phrases in `source.json`; uncertain labels stay unchanged and must be resolved before review can pass.

Review the runnable draft, not just the source catalog. Check the API contract, command bindings, help, examples and fixtures:

- Ordinary JSON fixtures and HTTP response fixtures must match captured source evidence. Missing or changed fixture data blocks review.
- Examples must add something beyond the Usage line: concrete arguments, meaningful options, structured values or other behaviour. Examples that only repeat the command path, including unresolved path placeholders, are rejected.
- Fix mechanically assembled English descriptions, missing required options, undeclared options and values that do not match their parameter types.
- Generate recommendation help only when the target resolves to a tracked, promoted CLI command. Otherwise, retain the unresolved evidence in `source.json`. Cross-plugin references are soft dependencies at load time, but release checks require an exact, non-deprecated target and reject recommendation cycles.

If an API has a clear response contract but no usable successful response example, set `fixture_unavailable` with the reason. Keep the command, response validation and original evidence; do not invent a success fixture. The command cannot use `--offline` or `--fixture`, and the missing example cannot serve as waiter evidence. For downloads, declare the documented success `media_type` on the response variant so an HTTP-200 error envelope is not saved as a file.

#### 3. Check lifecycle waiters

At each review, assess whether asynchronous operations have a safe retrieval command with documented terminal states. Bind a waiter only to a non-dangerous, explicitly retryable retrieval operation. Record missing state evidence or resource-identifying inputs as gaps instead of guessing when an operation is complete.

Keep waiter definitions and their upstream evidence in the catalog's `waiters` map. Review rejects waiter definitions that differ from the source catalog; promotion preserves accepted definitions. If you change the definitions, regenerate and review the draft before promotion. Test the real bundle offline for response shape, terminal outcomes, and command-specific help and completion.

<details>
<summary>Waiter metadata reference</summary>

| Field                              | Requirement                                                                                                                              |
|------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| `commands`                         | Explicit bindings to exact command IDs                                                                                                   |
| `path` or `xml_path`               | JSON state path, or an absolute namespace-aware XML path selecting exactly one scalar element; XML paths cannot use collection selectors |
| `success`, `failure`               | Documented terminal values; an empty failure value means no failure condition is documented                                              |
| `success_values`, `failure_values` | Additional documented terminal values when needed                                                                                        |
| `max_attempts`, `interval_seconds` | Positive attempt limit and polling interval in seconds                                                                                   |
| `evidence`                         | Source and meaning of the documented states                                                                                              |

For collection responses, add a `selector`: `path` locates the collection, `key` identifies each row, and `value` references an existing `$arg.<name>` or scalar string/integer `$param.<name>`. The state path is relative to the selected row. Check every captured row, preserve exact numeric identities, and keep null or unknown states pending. If a unique identity input is unavailable, record the blocker.

</details>

#### 4. Promote the accepted draft

Before promotion, confirm that review passes and the declared quality (`generated`, `reviewed` or `curated`) accurately describes the plugin.

Generation derives `source_fingerprint` from `source.json`. For existing plugins, it preserves the promoted version, channel, quality and core compatibility range, raising the minimum core version when required:

| Metadata feature                                    | Minimum core version |
|-----------------------------------------------------|----------------------|
| Waiters                                             | `0.5.0`              |
| Independent request-header, query and body bindings | `0.5.1`              |

Regeneration must not lower an existing compatibility requirement or reset the plugin version, channel or quality. Promotion updates plugin metadata and advances `baseline.json` to the accepted source:

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
- Reuse an output directory containing the current published indexes and plugin archives. When a plugin's archive contents are unchanged, packaging retains its existing compressed bytes and checksum across Go toolchain updates.
- After updating fixed release assets, keep archives referenced by the current signed indexes, index signatures, and core installation scripts, and remove old archives that are no longer referenced; prerelease channel archives should stay when the index still advertises them.

The plugin index and archives at the output root are for GitHub. Use `gitee/index.json` and `gitee/index.sig` to replace the same-named files on Gitee's fixed `plugins` Release. `gitee/releases.json` is a publication manifest, not a user download: it records each archive's plugin, version, channel, target tag, checksum, and expected download URL. Publish every listed immutable Gitee version Release and archive before replacing the fixed `plugins` index.

Core and plugin versions must follow Semantic Versioning 2.0.0. Do not prefix release versions with `v`. Use versions like `0.1.0-alpha.1` with the `alpha`/`beta` channels for pre-releases, and versions like `0.1.0` with the `stable` channel for stable releases. The defaults in `internal/version/version.go` only identify unpackaged development builds, and release packaging overrides the actual version and channel.

Signed indexes also include a `versions` map derived from the final merged entries: `versions.<channel>` for core and `versions.<name>.<channel>` for plugins, each containing the highest semantic version for that channel. README dynamic badges read these maps; the CLI continues to select downloads from the existing arrays.

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
go run ./tools/release --version 0.5.1 --channel stable --out ./dist/releases --gitee-plugin-download-root "https://gitee.com/ArvinZJC/ctyun-cli/releases/download" --platform "$(go env GOOS)/$(go env GOARCH)"
```

For real releases, GitHub remains the canonical source and CI artifact authority, while Gitee is the synchronised mirror for more reliable access from mainland China. The installed `ctyun` verifies hosted core and plugin indexes using the signing public key and checks archive SHA-256 hashes; the bootstrap scripts use the installation trust model described above.

## Related projects

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli): another unofficial CTyun CLI, written in Python, useful as a reference for users who prefer the Python ecosystem.
