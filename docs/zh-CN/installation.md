# 安装、更新与卸载 ctyun

[项目首页](../../README.md) · [English](../en/installation.md)

## 使用前须知

- **与官方 CLI 的关系**：天翼云已于 2026 年 7 月 2 日发布官方 `ctyun-cli`。本项目是独立维护的非官方实现。[天翼云产品动态](https://www.ctyun.cn/document/11095072/11096774)

  两个工具可以安装在同一个 Shell 环境中。运行 `ctyun-cli` 调用官方工具，运行 `ctyun` 调用本项目。两者均可从环境变量 `CTYUN_AK` 和 `CTYUN_SK` 读取凭据。详见[官方 CLI 指南](https://www.ctyun.cn/document/11095072/11096343)。

  如需在 Go 程序中调用 API，请查阅对应云产品的 SDK 文档。例如，天翼云在[关系数据库 MySQL 版的 SDK 参考](https://www.ctyun.cn/document/10033813/11065649)中提供了 `ctyun-go-sdk.zip` 下载。

  本项目重点维护终端交互、便于脚本组合处理的输出，以及可独立维护的插件。待官方 CLI 能够稳定支持这些工作流程时，我们会重新评估本项目的定位。

- 开通要管理的服务，并准备具有相应权限的 AK/SK 凭据。

- 本工具支持自研池（一类节点）的客户侧（C 端）接口，不支持业务／运营侧（B 端）接口和合营池（二类节点）。这里的节点分类不等同于 Redis 等产品文档中的Ⅰ／Ⅱ类资源池版本，不能仅凭编号判断接口是否属于本项目范围。

- 服务的具体使用条件请参阅[天翼云 OpenAPI 文档](https://eop.ctyun.cn/ebp/ctapiDocument/index)。

## 安装

安装脚本支持识别 AMD64 和 ARM64，所选通道须提供适用于当前操作系统和架构的归档。下方 Bash 示例使用 `curl` 下载安装脚本。脚本运行时需要 Bash、`awk`、`tar`，以及 `curl` 或 `wget`、`sha256sum` 或 `shasum`。在 Windows 上，请确保可通过 Path 调用 `tar.exe`。

以下脚本可安装原生 `ctyun` 可执行文件。脚本依次检查 `stable`、`beta`、`alpha` 三个通道，选择第一个提供当前平台安装包的通道。要指定通道，可设置 `CTYUN_INSTALL_CHANNEL`。要从 Gitee 下载安装脚本和发布包，请将脚本 URL 中的 `github.com` 替换为 `gitee.com`，并在运行脚本前将 `CTYUN_INSTALL_SOURCE` 设为 `gitee`。

macOS、Linux 和 WSL：

```sh
(
  installer=$(mktemp) || exit 1
  trap 'rm -f "$installer"' EXIT
  curl -fsSL https://github.com/ArvinZJC/ctyun-cli/releases/download/core/install.sh -o "$installer" && bash "$installer"
)
```

Windows PowerShell：

```powershell
& {
    Invoke-RestMethod https://github.com/ArvinZJC/ctyun-cli/releases/download/core/install.ps1 -ErrorAction Stop | Invoke-Expression
}
```

如果不确定当前终端是否为 PowerShell，请先从开始菜单或 Windows Terminal 的标签页菜单打开 Windows PowerShell，再运行 `$PSVersionTable.PSVersion` 确认；能看到版本信息后，在同一个窗口运行上面的安装命令。

引导安装脚本从选定的发布源下载索引和归档（默认源使用 HTTPS），并根据索引中的 SHA-256 校验归档。脚本不验证 `core-index.sig`，因此首次安装依赖所下载脚本及发布源的可信性。安装完成后，`ctyun` 在核心更新和托管插件操作中同时验证索引签名与归档哈希。

安装脚本支持这些环境变量：

| 变量                    | 用途                                                                                                         |
|-------------------------|--------------------------------------------------------------------------------------------------------------|
| `CTYUN_INSTALL_CHANNEL` | 固定安装通道，可设为 `stable`、`beta` 或 `alpha`                                                             |
| `CTYUN_INSTALL_SOURCE`  | 固定安装源，可设为 `auto`、`github` 或 `gitee`                                                               |
| `CTYUN_INSTALL_DIR`     | 覆盖安装目录；默认 macOS、Linux 和 WSL 为 `$HOME/.local/bin`，Windows 为 `%LOCALAPPDATA%\Programs\ctyun-cli` |

### 安装完成后

如果安装脚本输出了设置 `PATH` 的命令，请将它复制到准备使用 `ctyun` 的终端中执行。该命令会使用实际安装目录，包括通过 `CTYUN_INSTALL_DIR` 指定的目录。要让设置对以后打开的终端也生效，Zsh 用户可将输出的 `export` 命令加入 `~/.zshrc`，Bash 用户可将其加入自己的 Bash 启动文件。Windows 用户可在“环境变量”中将安装目录加入**用户变量**的 Path，然后重新打开 PowerShell。

先确认版本号与安装成功消息中的版本一致，再查看命令概览。这两条命令可以离线运行，无需配置凭据。

```sh
ctyun --version
ctyun --help
```

产品命令由单独安装的插件提供。以下命令可在线浏览插件列表、安装资源池插件并查看其命令帮助：

```sh
ctyun plugin list --available --channel all
ctyun plugin install region
ctyun region list --help
```

### 安装问题排查

- **找不到命令或版本不符**：在 Bash 或 Zsh 中运行 `type -a ctyun`，在 PowerShell 中运行 `Get-Command ctyun -All`，将第一项与安装脚本输出的可执行文件路径对照。也可以通过完整路径直接运行已安装的程序；在 PowerShell 中，如果路径用引号括起，请在前面加上调用运算符 `&`。
- **找不到可用版本**：检查报错中的操作系统、架构和通道。要恢复自动选择通道，请在 Bash 或 Zsh 中运行 `unset CTYUN_INSTALL_CHANNEL`，或在 PowerShell 中运行 `Remove-Item Env:CTYUN_INSTALL_CHANNEL -ErrorAction SilentlyContinue`，然后重新运行安装脚本。
- **下载或校验失败**：检查能否访问所选发布源，然后重试。如需切换到本项目的另一个镜像，请替换上方安装命令中的主机名，并将 `CTYUN_INSTALL_SOURCE` 设为对应的发布源。如果仍然校验失败，请先反馈版本号及发布源。
- **无法写入目标位置**：如果复制失败，请检查安装目录的可用空间。Bash 安装脚本会保留原有可执行文件，直到新文件准备就绪后才替换。如果没有目录写权限，或者可执行文件的名称已被目录占用，请换用其他安装目录。例如，在 Bash 或 Zsh 中运行 `export CTYUN_INSTALL_DIR="$HOME/ctyun-bin"`，或在 PowerShell 中运行 `$env:CTYUN_INSTALL_DIR = Join-Path $HOME 'ctyun-bin'`，然后重新运行安装脚本。

## 核心更新

使用 `ctyun update --check` 检查核心更新，使用 `ctyun upgrade` 安装更新。指定 `--source auto` 时，CLI 先访问 GitHub；如果无法获取索引或签名，或者签名验证失败，则尝试 Gitee。CLI 下载所选索引引用的归档并验证其 SHA-256 校验和。可用 `--source github` 或 `--source gitee` 明确指定下载源。如果归档下载失败，可指定另一个源重新运行命令。发布版默认检查自身所在的通道。可用 `--channel` 检查其他通道，升级时只安装比当前可执行文件更新的版本。

在 Windows 上，升级成功后，`ctyun.exe` 同级目录下的 `.ctyun.exe.previous` 会保留上一版可执行文件。下次升级时，CLI 会先校验这份备份再替换它。如果旧进程仍在运行，请关闭后重试。如果备份目录内有无法识别的文件，请先将其移出再重试。

```sh
ctyun update --check --source auto
ctyun upgrade --source auto
ctyun upgrade --source auto --channel alpha
```

## 卸载

卸载可执行文件前，可先列出已安装的插件，选择需要移除的插件。示例中的第二条命令会在确认后移除两个指定的插件。脚本可用 `--yes` 或 `-y` 跳过确认。

```sh
ctyun plugin list
ctyun plugin remove ecs region
```

如需删除所有已安装插件，运行 `ctyun plugin remove --all` 并按提示确认。

如需删除当前配置文件并在同一目录保留备份，可运行：

```sh
ctyun config reset
```

在 macOS、Linux 和 WSL 上，安装脚本默认将 `ctyun` 安装到 `$HOME/.local/bin`。如果选择了其他目录，请先将 `CTYUN_INSTALL_DIR` 设为该目录的绝对路径，再执行删除命令。

```sh
rm -f -- "${CTYUN_INSTALL_DIR:-$HOME/.local/bin}/ctyun"
```

Windows PowerShell 默认安装到 `%LOCALAPPDATA%\Programs\ctyun-cli\ctyun.exe`；如果安装时设置过 `CTYUN_INSTALL_DIR`，请使用同一个目录：

```powershell
$InstallDir = if ($env:CTYUN_INSTALL_DIR) { $env:CTYUN_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\ctyun-cli" }
$Target = Join-Path $InstallDir "ctyun.exe"
if (Test-Path -LiteralPath $Target -PathType Leaf) {
    Remove-Item -LiteralPath $Target -Force -ErrorAction Stop
}
```
