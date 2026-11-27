# Installing, updating and removing ctyun

[Project home](../../README-EN.md) · [简体中文](../zh-CN/installation.md)

## Before you use it

- **Relationship to the official CLI:** CTyun released the official `ctyun-cli` on 2 July 2026. This project is an independently maintained unofficial implementation. [CTyun release history](https://www.ctyun.cn/document/11095072/11096774)

  You can install both tools in the same shell. Run `ctyun-cli` for the official tool and `ctyun` for this project. Both can read credentials from `CTYUN_AK` and `CTYUN_SK`. See the [official CLI guide](https://www.ctyun.cn/document/11095072/11096343).

  For Go applications, consult the SDK documentation for the product you need. For example, CTyun publishes `ctyun-go-sdk.zip` in its [RDS for MySQL SDK reference](https://www.ctyun.cn/document/10033813/11065649).

  Development focuses on interactive terminal use, output that scripts can combine, and independently maintained plugins. We will reassess the project's role when the official CLI reliably supports the same workflows.

- Activate the service you want to manage and prepare an AK/SK credential pair with the required permissions.

- This CLI supports customer-side (C-side) APIs for self-operated resource pools (first-class nodes). Business/operations-side (B-side) APIs and joint-operation pools (second-class nodes) are outside its scope. This node classification is distinct from product-specific Type I/II resource-pool versions, such as those in Redis documentation; the numeral alone does not determine whether an API is in scope.

- Check the [CTyun OpenAPI documentation](https://eop.ctyun.cn/ebp/ctapiDocument/index) for service-specific requirements.

## Installation

The installers recognise AMD64 and ARM64. The selected channel must contain an archive for your operating system and architecture. The Bash instructions use `curl` to fetch the script. The script itself needs Bash, `awk`, `tar`, either `curl` or `wget`, and either `sha256sum` or `shasum`. On Windows, make sure `tar.exe` is on your Path.

The scripts below install the native `ctyun` binary. They select the first channel with a build for your platform, checking `stable`, `beta`, then `alpha`. Set `CTYUN_INSTALL_CHANNEL` to select a specific channel. To download both the installer and the release from Gitee, replace `github.com` in the installer URL with `gitee.com` and set `CTYUN_INSTALL_SOURCE` to `gitee` before running it.

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

If you are not sure whether the current terminal is PowerShell, open Windows PowerShell from the Start menu or the Windows Terminal tab menu, then run `$PSVersionTable.PSVersion` to confirm. After it prints version information, run the installation command above in the same window.

The bootstrap scripts download the index and archive from the selected release source (the default sources use HTTPS) and verify the archive against the SHA-256 hash in that index. They do not verify `core-index.sig`; bootstrap therefore trusts the downloaded script and release source. After installation, `ctyun` verifies signed indexes as well as archive hashes for core updates and hosted plugin operations.

The installation scripts support these environment variables:

| Variable                | Purpose                                                                                                                                          |
|-------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------|
| `CTYUN_INSTALL_CHANNEL` | Pin the installation channel to `stable`, `beta`, or `alpha`                                                                                     |
| `CTYUN_INSTALL_SOURCE`  | Pin the installation source to `auto`, `github`, or `gitee`                                                                                      |
| `CTYUN_INSTALL_DIR`     | Override the installation directory; defaults to `$HOME/.local/bin` on macOS, Linux, and WSL, and `%LOCALAPPDATA%\Programs\ctyun-cli` on Windows |

### After installation

If the installer prints a `PATH` command, copy it into the terminal where you want to use `ctyun`. It uses the directory you actually installed to, including a custom `CTYUN_INSTALL_DIR`. For future terminals, add the printed `export` line to `~/.zshrc` (Zsh) or your Bash startup file. On Windows, add the installation directory to your **user** Path in Environment Variables, then open a new PowerShell window.

Check that the version matches the installer’s success message and open the command overview. Both commands work without credentials or network access.

```sh
ctyun --version
ctyun --help
```

Product commands come from separately installed plugins. To browse the online registry, install the Region plugin, and read its command help, run:

```sh
ctyun plugin list --available --channel all
ctyun plugin install region
ctyun region list --help
```

### Troubleshooting installation

- **Command unavailable or wrong version.** Run `type -a ctyun` in Bash or Zsh, or `Get-Command ctyun -All` in PowerShell. Compare the first match with the path printed by the installer. You can also run the installed executable directly using its full path. In PowerShell, use the call operator `&` before a quoted executable path.
- **Release unavailable.** Check the reported OS, architecture, and channel. To restore automatic channel selection, run `unset CTYUN_INSTALL_CHANNEL` in Bash or Zsh, or `Remove-Item Env:CTYUN_INSTALL_CHANNEL -ErrorAction SilentlyContinue` in PowerShell, then rerun the installer.
- **Download or checksum failure.** Check access to the selected release source and retry. To use the other project mirror, change the hostname in the installation command above and set `CTYUN_INSTALL_SOURCE` to the corresponding source. If the checksum still fails, report the version and source before installing.
- **Destination unavailable.** If copying fails, check free space in the installation directory. The Bash installer keeps the previous executable until the replacement is ready. If the directory is unwritable or the executable name is already a directory, choose another install location. For example, run `export CTYUN_INSTALL_DIR="$HOME/ctyun-bin"` in Bash or Zsh, or `$env:CTYUN_INSTALL_DIR = Join-Path $HOME 'ctyun-bin'` in PowerShell, then rerun the installer.

## Core updates

Use `ctyun update --check` to check for a core update or `ctyun upgrade` to install it. With `--source auto`, the CLI tries GitHub first and tries Gitee if it cannot fetch the index and signature or verify the signature. It downloads the archive referenced by the selected index and verifies its SHA-256 checksum. To choose the download source explicitly, use `--source github` or `--source gitee`. If an archive download fails, rerun with the other source. A release build checks its own channel by default. Use `--channel` to check another channel. An upgrade installs only a version newer than the running binary.

On Windows, a successful upgrade keeps the previous executable beside `ctyun.exe` in `.ctyun.exe.previous`. The next upgrade verifies that copy before replacing it. If it is still running, close the older process and retry. If the backup directory contains unrecognised files, move them aside before retrying.

```sh
ctyun update --check --source auto
ctyun upgrade --source auto
ctyun upgrade --source auto --channel alpha
```

## Uninstallation

Before uninstalling the binary, list your plugins and choose which ones to remove. The example removes two named plugins after confirmation. Scripts can use `--yes` or `-y` to skip the prompt.

```sh
ctyun plugin list
ctyun plugin remove ecs region
```

To remove every installed plugin, run `ctyun plugin remove --all` and confirm at the prompt.

To remove the active configuration and keep a backup beside it, run:

```sh
ctyun config reset
```

On macOS, Linux, and WSL, the installer places `ctyun` in `$HOME/.local/bin` by default. If you chose another directory, set `CTYUN_INSTALL_DIR` to its absolute path before running the removal command.

```sh
rm -f -- "${CTYUN_INSTALL_DIR:-$HOME/.local/bin}/ctyun"
```

Windows PowerShell installs to `%LOCALAPPDATA%\Programs\ctyun-cli\ctyun.exe` by default. If you set `CTYUN_INSTALL_DIR` during installation, use the same directory:

```powershell
$InstallDir = if ($env:CTYUN_INSTALL_DIR) { $env:CTYUN_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\ctyun-cli" }
$Target = Join-Path $InstallDir "ctyun.exe"
if (Test-Path -LiteralPath $Target -PathType Leaf) {
    Remove-Item -LiteralPath $Target -Force -ErrorAction Stop
}
```
