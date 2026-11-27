# Finding and managing plugins

[Project home](../../README-EN.md) · [简体中文](../zh-CN/plugins.md)

A fresh `ctyun` installation includes the core commands. You can browse and install product plugins before configuring cloud credentials.

## Choose the plugin and channel

Look up the product’s command prefix and channel in the [product catalogue](catalogue.md). For example, Region is on `stable`, while ECS is on `beta`. To see what is published on the selected mirror, run `ctyun plugin list --available --channel all`. To see what this user already has installed, run `ctyun plugin list`. Installed bundles normally live in `~/.ctyun/plugins`.

## Install, repair or update a plugin

```sh
ctyun plugin search ecs --source auto --channel beta
ctyun plugin list --available --source auto
ctyun --lang en-GB plugin list --available --cols Plugin,Quality,Status --filter Status=available --source auto
ctyun plugin install region --source auto
ctyun plugin install ecs --source auto --channel beta
ctyun plugin list
```

Plugin management commands share these behaviours:

- `ctyun plugin search`, `ctyun plugin list --available`, `ctyun plugin install`, `ctyun plugin reinstall`, and `ctyun plugin update` support `--source` and `--channel`.
- `ctyun plugin list --available` shows hosted plugins with local installation status.
- `ctyun plugin list --available` and `ctyun plugin search` inspect the `stable` channel by default, and can use `--channel all` to inspect every registry channel.
- Install, reinstall, update, and update checks select the `stable` channel by default; choose prerelease plugins explicitly with `--channel beta` or `--channel alpha`.
- `ctyun plugin search` supports fuzzy matching and follows the table/JSON output controls.
- `ctyun plugin install` installs only absent plugins; it skips an installed plugin and never upgrades, downgrades, or replaces it through installation.
- `ctyun plugin reinstall` replaces an installed plugin with the version from the selected source and channel, even if that version is the same or older. With `--all`, it attempts to reinstall every installed plugin and reports a failure for each one missing from that channel. If your plugins come from different channels, name the plugins to reinstall in each channel.
- `ctyun plugin update` installs only versions with higher SemVer precedence.
- Install, reinstall, update, removal, and core upgrade show progress on stderr in an interactive terminal, then write one summary to stdout; redirected and piped runs emit no progress control sequences.

```sh
ctyun plugin reinstall region --source auto
ctyun plugin reinstall ecs --source auto --channel beta
ctyun plugin update --all --source auto
ctyun plugin update --all --source auto --channel beta
```

To install every absent plugin on the selected channel, use `ctyun plugin install --all` with that channel. To remove selected plugins, run `ctyun plugin remove ecs region` and confirm at the prompt. Add `--yes` in a script when removal is intended.

## Check updates or diagnose a missing command

- Run `ctyun plugin list --updates --channel beta` to check for newer beta plugins. Apply an individual update with `ctyun plugin update ecs --channel beta`.
- If a product is missing from search, repeat the search with `--channel all`. If its command is missing from help, check the local plugin list and install the product from the channel shown in the catalogue.
- If installation reports a core compatibility error, check `ctyun --version` and `ctyun update --check`. If fetching the index fails, run `ctyun doctor network --source github` or select `gitee` to test that mirror.
