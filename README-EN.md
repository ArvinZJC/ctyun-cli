# ctyun-cli

[![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fcore%2Fcore-index.json&query=%24.versions.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/core)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/ArvinZJC/ctyun-cli)
[![GitHub License](https://img.shields.io/github/license/ArvinZJC/ctyun-cli?label=licence)](./LICENCE)

[简体中文](./README.md) | English

`ctyun` is an unofficial Go command-line tool for querying and managing CTyun resources. Install the plugins you need, inspect results in a table or request JSON for scripts. The repository and package are called `ctyun-cli`. CTyun’s official CLI uses the command `ctyun-cli`; the two can coexist.

The project covers customer-side APIs for self-operated resource pools (first-class nodes). Core help, errors, runtime warnings, plugin names, command descriptions and table labels are available in Chinese and English.

## Installation

macOS, Linux, and WSL:

```sh
(
  installer=$(mktemp) || exit 1
  trap 'rm -f "$installer"' EXIT
  curl -fsSL https://github.com/ArvinZJC/ctyun-cli/releases/download/core/install.sh -o "$installer" && bash "$installer"
)
```

Windows PowerShell:

```powershell
& {
    Invoke-RestMethod https://github.com/ArvinZJC/ctyun-cli/releases/download/core/install.ps1 -ErrorAction Stop | Invoke-Expression
}
```

If the installer prints a `PATH` command, run it in the terminal where you will use `ctyun`. For Gitee downloads, a custom installation directory or troubleshooting, see the [installation guide](docs/en/installation.md).

## First query

Check the binary and install the Region plugin. Installing a plugin requires network access. You can read its help before setting up a cloud account.

```sh
ctyun --version
ctyun plugin install region
ctyun region list --help
```

Once your shell has `CTYUN_AK` and `CTYUN_SK` for an account with permission to list resource pools, run:

```sh
ctyun region list
```

The table shows the resource pools visible to your account. Use `ctyun region list --output json` for the API response. See [credentials and profiles](docs/en/configuration.md) for setup.

## Documentation

| I want to… | Guide |
|---|---|
| Install, upgrade or uninstall the CLI | [Installation](docs/en/installation.md) |
| Configure credentials, select a profile or diagnose a connection | [Configuration](docs/en/configuration.md) |
| Find and update plugins, or check product coverage | [Plugin management](docs/en/plugins.md) · [Product catalogue](docs/en/catalogue.md) |
| Read command help, filter tables, script queries or wait for a resource | [Running commands](docs/en/commands.md) |
| Upload or download files using storage credentials | [Storage and file transfers](docs/en/storage.md) |

### For contributors

- [Core development](docs/en/development.md) covers the request path, where to make changes, offline testing and the required checks.
- [Plugin authoring](docs/en/plugin-authoring.md) follows one command from source evidence to a reviewed bundle.
- [Release maintenance](docs/en/releases.md) covers versioning, installer validation, signing and publication on GitHub and Gitee.

If you are reading this README from a binary archive, the full guides are available in the [source repository](https://github.com/ArvinZJC/ctyun-cli).

## Related projects

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli): another unofficial CTyun CLI, written in Python, useful as a reference for users who prefer the Python ecosystem.
