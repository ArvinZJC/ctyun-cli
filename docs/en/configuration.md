# Credentials, profiles and diagnostics

[Project home](../../README-EN.md) · [简体中文](../zh-CN/configuration.md)

Live requests prefer AK/SK from the process environment:

```sh
export CTYUN_AK=...
export CTYUN_SK=...
```

For each missing environment value, `ctyun` reads the corresponding `ak` or `sk` from the selected profile, then from the top-level config. If a live request uses either stored credential, `ctyun` warns on stderr. Set `CTYUN_WARN_CONFIG_CREDENTIALS=0` to suppress the warning for the process. To disable it by default, run `ctyun config set warn_config_credentials false`. If a profile enables the warning, add `--profile <name>` to change the setting in that profile.

## Configuration and profiles

The config file is selected by `--config`, then `CTYUN_CONFIG`, then `~/.ctyun/config.json`. `--profile` overrides `active_profile`. With neither set, the CLI selects the only configured profile. If several profiles exist, choose one with `--profile` or set an active profile.

The base region, endpoint override, and request timeout come from the selected profile. To see which source supplies a setting, run `ctyun config explain {key}`.

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
```

If you intend to keep credentials in this profile, read each secret from standard input. These commands copy values already present in the current shell into the configuration file.

```sh
printf '%s\n' "$CTYUN_AK" | ctyun config profile set-secret prod ak --from-stdin
printf '%s\n' "$CTYUN_SK" | ctyun config profile set-secret prod sk --from-stdin
```

`ctyun config show` displays stored JSON and masks saved AK/SK values like `aa*****dd`; unset values are omitted. `ctyun config explain` instead reports effective base settings and the source that won for each value. Sensitive rows report only whether a value is configured and never reveal, mask, fingerprint, or otherwise derive AK/SK.

Use `ctyun doctor local` to check configuration, credentials, resource pool settings, and installed plugins without network requests or local changes. It reports all independent findings and exits with status 1 if any check fails; warnings and skipped checks exit with status 0. Use `ctyun doctor network` for online source and API endpoint diagnostics.

After confirmation, `ctyun config reset` backs up the configuration beside the original file, then deletes the original. It uses the first available suffix starting with `.bak`, then `.bak.2`, `.bak.3`, and so on. Scripts can use `--yes` or `-y` to skip the prompt.

If another `ctyun` process changes the configuration during a save or reset, the command can report a conflict. Check the current settings with `ctyun config show` before retrying.

## Language

Supported languages are `zh-CN`, `en-US`, and `en-GB`. Language resolution is `--lang`, then `CTYUN_LANGUAGE`, then profile `language`, then the OS locale. If nothing matches, `zh-CN` is used.

## Work with two profiles

Use a separate profile for each resource pool or environment you regularly address. Substitute a region ID visible to your account. These commands create or edit the named profile, select it for a single inspection, then make it the default for subsequent commands.

```sh
ctyun config profile set staging region 81f7728662dd11ec810800155d307d5b
ctyun config profile list
ctyun --profile staging config explain region
ctyun config profile use staging
```

If the credentials appear to belong to a different account after switching profiles, run `ctyun --profile staging config explain ak` and repeat for `sk`. An environment value still takes precedence over a profile value. For Windows PowerShell the environment syntax is `$env:CTYUN_AK` and `$env:CTYUN_SK`. Provision secrets through the mechanism you use for the shell or CI job.

## Diagnose the failing layer

| Symptom | What to check |
|---|---|
| Unexpected region, profile or config file | Run `ctyun config path` and `ctyun config explain` in the shell that runs the failing command |
| Local settings or an installed bundle are invalid | Run `ctyun doctor local` and inspect each failed check |
| The registry or API endpoint cannot be reached | Run `ctyun doctor network`. Use `--source github` or `--source gitee` to isolate a mirror |
| The service responds but denies a query | Confirm the credential source, the account’s permission for the operation and the service’s availability in that resource pool |
