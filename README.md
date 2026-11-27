# ctyun-cli

[![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fcore%2Fcore-index.json&query=%24.versions.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/core)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/ArvinZJC/ctyun-cli)
[![GitHub License](https://img.shields.io/github/license/ArvinZJC/ctyun-cli?label=licence)](./LICENCE)

简体中文 | [English](./README-EN.md)

`ctyun` 是使用 Go 编写的非官方命令行工具，通过天翼云 API 查询和管理云资源。它采用产品插件，注重终端使用体验，提供便于查看的表格和适合脚本处理的 JSON 输出。仓库和软件包名为 `ctyun-cli`。

[安装](#安装) · [配置](#鉴权配置与语言) · [插件](#插件) · [使用](#使用命令) · [更新](#核心更新) · [存储](#存储鉴权与文件传输) · [参与开发](#开发者与贡献者工作流)

## 亮点

- 默认表格输出，支持中英文内容宽度对齐。
- `--output json` 便于脚本和其他工具处理结果。
- 产品插件可以独立安装和更新。
- 支持等待资源或任务状态，并在危险操作前提示确认。
- 支持中英文的核心帮助、错误提示、运行时提醒、插件名称、命令说明和表格列名。

## 使用前须知

- **与官方 CLI 的关系**：天翼云已于 2026 年 7 月 2 日发布官方 `ctyun-cli`。本项目是独立维护的非官方实现。

  两个工具可以安装在同一个 Shell 环境中。运行 `ctyun-cli` 调用官方工具，运行 `ctyun` 调用本项目。两者均可从环境变量 `CTYUN_AK` 和 `CTYUN_SK` 读取凭据。详见[官方 CLI 指南](https://www.ctyun.cn/document/11095072/11096343)。

  如需在 Go 程序中调用 API，请查阅对应云产品的 SDK 文档。例如，天翼云在[关系数据库 MySQL 版的 SDK 参考](https://www.ctyun.cn/document/10033813/11065649)中提供了 `ctyun-go-sdk.zip` 下载。

  本项目重点维护终端交互、便于脚本组合处理的输出，以及可独立维护的插件。待官方 CLI 能够稳定支持这些工作流程时，我们会重新评估本项目的定位。

- 开通要管理的服务，并准备具有相应权限的 AK/SK 凭据。

- 本工具支持自研池（一类节点）的客户侧（C 端）接口，不支持业务／运营侧（B 端）接口和合营池（二类节点）。这里的节点分类不等同于 Redis 等产品文档中的Ⅰ／Ⅱ类资源池版本，不能仅凭编号判断接口是否属于本项目范围。

- 服务的具体使用条件请参阅[天翼云 OpenAPI 文档](https://eop.ctyun.cn/ebp/ctapiDocument/index)。

## 安装

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
- **下载或校验失败**：检查能否访问所选发布源，然后重试。如需切换到另一个官方镜像，请替换上方安装命令中的主机名，并将 `CTYUN_INSTALL_SOURCE` 设为对应的发布源。如果仍然校验失败，请先反馈版本号及发布源。
- **无法写入目标位置**：如果复制失败，请检查安装目录的可用空间。Bash 安装脚本会保留原有可执行文件，直到新文件准备就绪后才替换。如果没有目录写权限，或者可执行文件的名称已被目录占用，请换用其他安装目录。例如，在 Bash 或 Zsh 中运行 `export CTYUN_INSTALL_DIR="$HOME/ctyun-bin"`，或在 PowerShell 中运行 `$env:CTYUN_INSTALL_DIR = Join-Path $HOME 'ctyun-bin'`，然后重新运行安装脚本。

## 鉴权、配置与语言

实时请求优先从进程环境读取 AK/SK：

```sh
export CTYUN_AK=...
export CTYUN_SK=...
```

如果某个凭据环境变量缺失，`ctyun` 会先从所选配置档案读取对应的 `ak` 或 `sk`，再回退到全局配置。实时请求只要使用了配置中的任一凭据，`ctyun` 就会向标准错误输出（stderr）写入提醒。设置 `CTYUN_WARN_CONFIG_CREDENTIALS=0` 可关闭当前进程的提醒。要默认关闭提醒，请运行 `ctyun config set warn_config_credentials false`。如果某个配置档案显式开启了提醒，可在该命令末尾加上 `--profile <name>` 来修改该档案的设置。

### 配置与配置档案

配置文件按 `--config`、`CTYUN_CONFIG`、`~/.ctyun/config.json` 的顺序选择；`--profile` 优先于 `active_profile`。两者都未设置时，如果仅有一个配置档案，CLI 会自动选用它。如果有多个，请用 `--profile` 指定一个，或设置当前配置档案。

默认资源池、自定义 API 端点和请求超时来自选定的配置档案。要查看某项设置的实际来源，请运行 `ctyun config explain {key}`。

常用环境变量：

| 变量                            | 用途                                                               |
|---------------------------------|--------------------------------------------------------------------|
| `CTYUN_CONFIG`                  | 覆盖配置文件路径                                                   |
| `CTYUN_AK`                      | 实时请求使用的天翼云 AK                                            |
| `CTYUN_SK`                      | 实时请求使用的天翼云 SK                                            |
| `CTYUN_LANGUAGE`                | 覆盖界面语言，可设为 `zh-CN`、`en-US` 或 `en-GB`                   |
| `CTYUN_WARN_CONFIG_CREDENTIALS` | 设为 `0` 可关闭使用配置中 AK/SK 时的提醒                           |
| `CTYUN_WARN_DEPRECATED`         | 设为 `0` 可关闭使用已弃用命令、选项或输出字段时的提醒              |
| `CTYUN_PLUGIN_SOURCE`           | 插件安装、搜索和更新的默认来源，可设为 `auto`、`github` 或 `gitee` |
| `CTYUN_UPGRADE_SOURCE`          | 核心更新的默认来源，可设为 `auto`、`github` 或 `gitee`             |

安全建议：

- 优先使用环境变量传入 AK/SK；如果写入配置文件，请不要提交到仓库，并限制文件权限。
- 不要把 AK/SK 写入脚本、命令历史或日志。
- 为 `ctyun` 使用最小权限的 IAM 用户 AK/SK，并定期轮换。
- 避免在共享机器或 CI 日志中暴露环境变量。
- 使用 `--debug` 排查请求时，分享日志前仍应再次检查敏感信息。

配置文件适合保存资源池、语言、超时、用于测试或私有环境的高级 API 终端节点覆盖、警告偏好，也可以作为 `CTYUN_AK`/`CTYUN_SK` 的后备来源。全局键包括 `active_profile`、`ak`、`sk`、`warn_config_credentials` 和 `warn_deprecated`；配置档案键包括 `region`、`language`、`endpoint_url`、`timeout_seconds`、`ak`、`sk`、`warn_config_credentials` 和 `warn_deprecated`，命令帮助会列出其值格式和固定默认值。

```json
{
  "profiles": {
    "prod": {
      "region": "81f7728662dd11ec810800155d307d5b"
    }
  }
}
```

可通过非交互命令查看和更新配置：

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

`ctyun config show` 显示已存储的 JSON，并把已保存的 AK/SK 显示为 `aa*****dd` 这样的掩码；未配置的值会被省略。`ctyun config explain` 则显示生效的基础设置，以及每个值最终采用的来源。敏感设置只说明是否已配置，不会显示、掩码、指纹化或以其他方式派生 AK/SK。

使用 `ctyun doctor local` 检查配置、凭据、资源池设置和已安装插件，不访问网络或修改本地状态。它会输出所有独立检查结果；存在失败项时退出码为 1，只有警告或跳过项时为 0。使用 `ctyun doctor network` 检查在线下载源和 API 端点。

确认后，`ctyun config reset` 会在配置文件所在目录创建备份，再删除原文件。备份依次尝试 `.bak`、`.bak.2`、`.bak.3` 等后缀，使用第一个未被占用的名称。脚本中可使用 `--yes` 或 `-y` 跳过提示。

如果另一个 `ctyun` 进程在保存或重置期间修改了配置，命令可能会提示冲突。请先运行 `ctyun config show` 查看当前配置，再决定是否重试。

### 语言

支持的语言为 `zh-CN`、`en-US` 和 `en-GB`。语言选择顺序为 `--lang`、`CTYUN_LANGUAGE`、配置档案中的 `language`、系统语言；无法匹配时默认 `zh-CN`。

## 插件

新安装的 `ctyun` 包含核心命令。你可以在配置云凭据之前浏览并安装所需的产品插件。

```sh
ctyun plugin search ecs --source auto --channel beta
ctyun plugin list --available --source auto
ctyun plugin list --available --cols 插件,质量,状态 --filter 状态=可安装 --source auto
ctyun plugin install region --source auto
ctyun plugin install ecs --source auto --channel beta
ctyun plugin install --all --source auto
ctyun plugin list
```

<details>
<summary>插件列表</summary>

| 名称                       | 插件                         | 产品                         | 版本                                                                                                                                                                                                                                                                                                                  | 通道     | 质量        | 命令 | 操作 |
|----------------------------|------------------------------|------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|-------------|-----:|-----:|
| 应用云主机 ACS             | `acs`                        | `acs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27acs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   15 |   15 |
| 智能体引擎                 | `agent-engine`               | `agent-engine`               | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27agent-engine%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)               | `beta`   | `generated` |   11 |   11 |
| AI 能力开放平台            | `ai-capability`              | `ai-capability`              | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ai-capability%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)              | `beta`   | `generated` |   41 |   41 |
| AI 安全                    | `ai-security`                | `ai-security`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ai-security%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |    5 |    5 |
| API 网关                   | `api-gateway`                | `api-gateway`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27api-gateway%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |  162 |  162 |
| 应用性能监控 APM           | `apm`                        | `apm`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27apm%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   63 |   63 |
| 应用加速                   | `application-acceleration`   | `application-acceleration`   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27application-acceleration%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)   | `beta`   | `generated` |   12 |   12 |
| 应用高可用                 | `application-ha`             | `application-ha`             | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27application-ha%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)             | `beta`   | `generated` |   43 |   43 |
| 弹性伸缩服务 AS            | `as`                         | `as`                         | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27as%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                         | `beta`   | `generated` |   63 |   63 |
| 账务                       | `billing`                    | `billing`                    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27billing%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                    | `beta`   | `generated` |   16 |   16 |
| 云应用引擎 CAE             | `cae`                        | `cae`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cae%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   44 |   44 |
| 云备份 CBR                 | `cbr`                        | `cbr`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cbr%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   22 |   22 |
| 云容器引擎 CCE             | `cce`                        | `cce`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cce%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |  308 |  308 |
| 分布式容器云平台 CCE One   | `cce-one`                    | `cce-one`                    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cce-one%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                    | `beta`   | `generated` |   41 |   41 |
| 云专线 CDA                 | `cda`                        | `cda`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cda%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   61 |   61 |
| 内容分发网络加速 CDN       | `cdn`                        | `cdn`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cdn%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   92 |   92 |
| 云容灾 CDR                 | `cdr`                        | `cdr`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cdr%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   31 |   31 |
| 证书管理服务               | `certificate`                | `certificate`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27certificate%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |   14 |   14 |
| 函数计算                   | `cf`                         | `cf`                         | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cf%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                         | `beta`   | `generated` |   67 |   67 |
| 对象存储（经典版）I型      | `classic-object-storage`     | `classic-object-storage`     | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27classic-object-storage%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)     | `beta`   | `generated` |  219 |  219 |
| 云数据库 ClickHouse        | `clickhouse`                 | `clickhouse`                 | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27clickhouse%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                 | `beta`   | `generated` |   55 |   55 |
| 云助手                     | `cloud-assistant`            | `cloud-assistant`            | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cloud-assistant%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)            | `beta`   | `generated` |   11 |   11 |
| 云审计                     | `cloud-audit`                | `cloud-audit`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cloud-audit%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |   13 |   13 |
| 云原生API网关              | `cloud-native-api-gateway`   | `cloud-native-api-gateway`   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cloud-native-api-gateway%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)   | `beta`   | `generated` |  186 |  186 |
| 云搜索服务                 | `cloud-search`               | `cloud-search`               | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cloud-search%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)               | `beta`   | `generated` |    7 |    7 |
| 云监控服务 CM              | `cm`                         | `cm`                         | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27cm%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                         | `beta`   | `generated` |  173 |  173 |
| 公共服务                   | `common`                     | `common`                     | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27common%27%5D.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                     | `stable` | `curated`   |    1 |    1 |
| 容器镜像服务 CRS           | `crs`                        | `crs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27crs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |  269 |  269 |
| 专属云（计算独享型）       | `dec`                        | `dec`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27dec%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |    9 |    9 |
| 数据管理服务               | `dms`                        | `dms`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27dms%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   78 |   78 |
| 物理机 DPS                 | `dps`                        | `dps`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27dps%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   60 |   60 |
| 分布式关系型数据库         | `drds`                       | `drds`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27drds%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |  125 |  125 |
| 数据传输服务 DTS           | `dts`                        | `dts`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27dts%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   72 |   72 |
| 云间高速（标准版） EC      | `ec`                         | `ec`                         | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ec%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                         | `beta`   | `generated` |   51 |   51 |
| 弹性容器实例 ECI           | `eci`                        | `eci`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27eci%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   37 |   37 |
| 天翼云电脑（政企版）       | `ecpc`                       | `ecpc`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ecpc%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |  371 |  371 |
| 弹性云主机 ECS             | `ecs`                        | `ecs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ecs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |  209 |  209 |
| 智能边缘云                 | `ecx`                        | `ecx`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ecx%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |  252 |  252 |
| DDoS 高防（边缘云版）      | `edge-ddos`                  | `edge-ddos`                  | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27edge-ddos%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                  | `beta`   | `generated` |   39 |   39 |
| 边缘安全加速平台           | `edge-security-acceleration` | `edge-security-acceleration` | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27edge-security-acceleration%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins) | `beta`   | `generated` |   79 |   79 |
| Web 应用防火墙（边缘云版） | `edge-waf`                   | `edge-waf`                   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27edge-waf%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                   | `beta`   | `generated` |   68 |   68 |
| 弹性高性能计算 E-HPC       | `ehpc`                       | `ehpc`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ehpc%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |   40 |   40 |
| 弹性IP EIP                 | `eip`                        | `eip`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27eip%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   57 |   57 |
| 弹性负载均衡 ELB           | `elb`                        | `elb`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27elb%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |  114 |  114 |
| 翼 MapReduce               | `emr`                        | `emr`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27emr%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   13 |   13 |
| 事件总线 EventBridge    | `eventbridge`                | `eventbridge`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27eventbridge%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `beta`   | `reviewed`  |      19 |      19 |
| 云硬盘 EVS                 | `evs`                        | `evs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27evs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   36 |   36 |
| 全局流量管理               | `gtm`                        | `gtm`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27gtm%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   21 |   21 |
| 高性能并行文件服务 HPFS    | `hpfs`                       | `hpfs`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27hpfs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |   40 |   40 |
| 分布式融合数据库 HTAP      | `htap`                       | `htap`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27htap%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |   47 |   47 |
| 慧聚一站式智算服务平台     | `huiju`                      | `huiju`                      | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27huiju%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `beta`   | `generated` |   86 |   86 |
| 统一身份认证               | `iam`                        | `iam`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27iam%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   91 |   91 |
| 镜像服务 IMS               | `ims`                        | `ims`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ims%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   27 |   27 |
| 时序数据库 Influx          | `influxdb`                   | `influxdb`                   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27influxdb%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                   | `beta`   | `generated` |   48 |   48 |
| 任务                       | `job`                        | `job`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27job%27%5D.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `stable` | `curated`   |    1 |    1 |
| 分布式消息服务 Kafka       | `kafka`                      | `kafka`                      | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27kafka%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `beta`   | `generated` |  125 |  125 |
| 密钥管理                   | `kms`                        | `kms`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27kms%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   82 |   82 |
| 视频直播                   | `live-streaming`             | `live-streaming`             | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27live-streaming%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)             | `beta`   | `generated` |   90 |   90 |
| 云日志服务 LTS             | `lts`                        | `lts`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27lts%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |  171 |  171 |
| 媒体存储                   | `media-storage`              | `media-storage`              | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27media-storage%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)              | `beta`   | `generated` |  124 |  124 |
| 模型训推                   | `model-training`             | `model-training`             | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27model-training%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)             | `beta`   | `generated` |   67 |   67 |
| 文档数据库服务             | `mongodb`                    | `mongodb`                    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27mongodb%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                    | `beta`   | `generated` |  180 |  180 |
| 分布式消息服务 MQTT        | `mqtt`                       | `mqtt`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27mqtt%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |   61 |   61 |
| 微服务云应用平台 MSAP      | `msap`                       | `msap`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27msap%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |  157 |  157 |
| 微服务引擎 MSE             | `mse`                        | `mse`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27mse%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |  243 |  243 |
| NAT网关                    | `nat`                        | `nat`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27nat%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   22 |   22 |
| 云防火墙（原生版）         | `native-firewall`            | `native-firewall`            | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27native-firewall%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)            | `beta`   | `generated` |  104 |  104 |
| 海量文件服务 OceanFS       | `oceanfs`                    | `oceanfs`                    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27oceanfs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                    | `beta`   | `generated` |   41 |   41 |
| 订单                       | `order`                      | `order`                      | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27order%27%5D.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `stable` | `curated`   |    7 |    7 |
| 内网DNS                    | `private-dns`                | `private-dns`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27private-dns%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |   40 |   40 |
| 私网NAT网关                | `private-nat`                | `private-nat`                | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27private-nat%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                | `beta`   | `generated` |   21 |   21 |
| 分布式消息服务 RabbitMQ    | `rabbitmq`                   | `rabbitmq`                   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27rabbitmq%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                   | `beta`   | `generated` |   39 |   39 |
| 关系数据库MySQL版          | `rds-mysql`                  | `rds-mysql`                  | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27rds-mysql%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                  | `beta`   | `generated` |  224 |  224 |
| 关系数据库 PostgreSQL 版   | `rds-postgresql`             | `rds-postgresql`             | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27rds-postgresql%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)             | `beta`   | `generated` |  156 |  156 |
| 关系型数据库 SQL Server    | `rds-sqlserver`              | `rds-sqlserver`              | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27rds-sqlserver%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)              | `beta`   | `generated` |  116 |  116 |
| 分布式缓存服务 Redis 版    | `redis`                      | `redis`                      | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27redis%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `beta`   | `generated` |  207 |  207 |
| 资源池                     | `region`                     | `region`                     | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27region%27%5D.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                     | `stable` | `curated`   |    7 |    7 |
| 科研助手                   | `research-assistant`         | `research-assistant`         | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27research-assistant%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)         | `beta`   | `generated` |   48 |   48 |
| 分布式消息服务 RocketMQ    | `rocketmq`                   | `rocketmq`                   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27rocketmq%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                   | `beta`   | `generated` |   46 |   46 |
| 资源编排 ROS               | `ros`                        | `ros`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27ros%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   40 |   40 |
| 天翼云 SD-WAN              | `sdwan`                      | `sdwan`                      | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27sdwan%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                      | `beta`   | `generated` |  138 |  138 |
| 服务器安全卫士（原生版）   | `server-security`            | `server-security`            | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27server-security%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)            | `beta`   | `generated` |  154 |  154 |
| 应用服务网格               | `service-mesh`               | `service-mesh`               | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27service-mesh%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)               | `beta`   | `generated` |   23 |   23 |
| 弹性文件服务 SFS           | `sfs`                        | `sfs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27sfs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   56 |   56 |
| 智能 DNS                   | `smart-dns`                  | `smart-dns`                  | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27smart-dns%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                  | `beta`   | `generated` |   19 |   19 |
| 云通信-短信                | `sms`                        | `sms`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27sms%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   24 |   24 |
| 星辰TokenHub运营服务平台   | `tokenhub`                   | `tokenhub`                   | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27tokenhub%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                   | `beta`   | `generated` |   19 |   19 |
| 共享流量包                 | `traffic-package`            | `traffic-package`            | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27traffic-package%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)            | `beta`   | `generated` |    6 |    6 |
| 云硬盘备份 VBS             | `vbs`                        | `vbs`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27vbs%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   37 |   37 |
| 智能视图服务               | `visual-service`             | `visual-service`             | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27visual-service%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)             | `beta`   | `generated` |   46 |   46 |
| 云点播                     | `vod`                        | `vod`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27vod%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   38 |   38 |
| 虚拟私有云 VPC             | `vpc`                        | `vpc`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27vpc%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |  207 |  207 |
| VPC终端节点                | `vpce`                       | `vpce`                       | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27vpce%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                       | `beta`   | `generated` |   47 |   47 |
| Web 应用防火墙（原生版）   | `waf`                        | `waf`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27waf%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |   94 |   94 |
| 全站加速                   | `whole-site-acceleration`    | `whole-site-acceleration`    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27whole-site-acceleration%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)    | `beta`   | `generated` |   96 |   96 |
| 云骁智算平台               | `yunxiao`                    | `yunxiao`                    | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27yunxiao%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                    | `beta`   | `generated` |   88 |   88 |
| 对象存储 ZOS               | `zos`                        | `zos`                        | [![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fplugins%2Findex.json&query=%24.versions%5B%27zos%27%5D.beta&label=release&color=orange)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/plugins)                        | `beta`   | `generated` |  106 |  106 |

质量字段表示插件元数据的整理程度：`generated` 表示工具生成的初稿，`reviewed` 表示已完成基础复核，`curated` 表示作为维护版本持续更新。

</details>

当前提供 97 个插件、7,751 条命令，覆盖自营资源池的客户侧 API。部分产品或接口因缺少可核实的公开服务地址、请求或响应说明而暂不支持；合营池和其他范围外接口不纳入支持。覆盖情况以已核实的官方文档为依据，不代表所有接口均已通过真实云环境验证。

核心 `0.5.2` 支持上表所有插件，其中 46 个插件要求核心版本至少为 `0.5.1`。插件版本和发布通道仍独立于核心版本维护。

使用以下插件时，请留意各自的适用范围、配置要求和功能限制：

- 网关：`api-gateway` 对应资源与订单代理接口目录，仅支持其中的自营客户侧用法；`cloud-native-api-gateway` 管理独立的云原生 API 网关实例，`mse` 中的网关命令属于微服务引擎。三者的服务地址和接口契约不同。

- 网络：`nat` 管理公网 NAT 网关，`private-nat` 管理私网 NAT 网关。暂不支持旧版 EIP 网络查询。
- RDS：暂不支持 6 个 PostgreSQL 下载或导出接口。已下线的 MySQL 跨地域备份目标地域查询命令仍保留弃用警告，但无法在线调用。
- Redis：`redis` 基于官方标为“2023-10-18 II类资源池”的 API 目录（`vid=270`）；该标签是 Redis 产品的目录分类，不代表本项目所排除的合营池（二类节点）。历史接口位于 `ctyun redis legacy`；要求核心版本 `>=0.5.1 <1.0.0`。所有命令均提供离线响应夹具，部分生命周期操作暂不支持等待完成。
- MongoDB：`mongodb` 的旧版接口位于 `ctyun mongodb v1`；要求核心版本 `>=0.5.1 <1.0.0`。三个导出 API 因二进制响应契约不明确暂未提供；所有已支持命令均提供离线响应夹具。
- 存储：`media-storage` 和 `classic-object-storage` 的 `native` 命令使用独立的存储凭证，详见[存储鉴权与文件传输](#存储鉴权与文件传输)。媒体存储 OpenAPI 网关适用于西藏资源池 1 区。
- 云助手：`ctyun cloud-assistant command run` 对应的 API 可用，但公开 API 文档目前不可用。

插件管理命令共享这些行为：

- `ctyun plugin search`、`ctyun plugin list --available`、`ctyun plugin install`、`ctyun plugin reinstall` 和 `ctyun plugin update` 都支持 `--source` 和 `--channel`。
- `ctyun plugin list --available` 会显示托管插件及本地安装状态。
- `ctyun plugin list --available` 和 `ctyun plugin search` 默认查看 `stable` 通道，也可使用 `--channel all` 查看所有插件源通道。
- 安装、重装、更新和更新检查默认选择 `stable` 通道；如需选择预发布插件，请显式指定 `--channel beta` 或 `--channel alpha`。
- `ctyun plugin search` 支持模糊搜索，并遵循表格/JSON 输出控制。
- `ctyun plugin install` 只安装尚未安装的插件；如果插件已安装，则跳过，不会通过 install 升级、降级或覆盖现有版本。
- `ctyun plugin reinstall` 使用所选来源和通道中的版本重装已安装的插件，即使该版本与当前版本相同或更低。指定 `--all` 时，命令会尝试重装所有已安装插件，在所选通道中找不到的插件会逐个计为失败。如果已安装的插件来自不同通道，请按通道分别指定要重装的插件名。
- `ctyun plugin update` 只安装 SemVer 优先级更高的版本。
- 安装、重装、更新、删除和核心升级在交互式终端中通过 stderr 显示进度，完成后只向 stdout 输出一条汇总；重定向或管道场景不会输出进度控制字符。

```sh
ctyun plugin reinstall region --source auto
ctyun plugin reinstall ecs --source auto --channel beta
ctyun plugin update --all --source auto
ctyun plugin update --all --source auto --channel beta
ctyun plugin remove ecs region --yes
```

## 使用命令

这些命令不依赖产品插件，适合安装后先确认版本、查看帮助、生成补全脚本或检查网络连通性：

```sh
ctyun --version
ctyun help
ctyun help config
ctyun completion zsh
ctyun doctor local
ctyun doctor network
```

插件命令的帮助会在安装对应插件后可用，例如 `ctyun help region list`。

### 产品命令

危险操作默认提示输入 `y/N` 确认；脚本中可使用 `--yes` 或 `-y` 跳过提示。

安装对应插件后，常用产品命令形态如下：

```sh
ctyun region list
ctyun region list --name 华东1 --cols "资源池 ID,资源池名称,地域编号"
ctyun ecs instance list --cols "实例 ID,名称,状态"
ctyun ecs instance list --name api-test01
ctyun ecs instance show c5a7966a-88e7-362b-6e11-c2d8fbfc07ca
```

### 输出与筛选

- `--cols`、`--filter` 和 `--sort` 可使用表格中看到的列名，也兼容稳定列键。列名有歧义时，可使用帮助中显示的完整列名或稳定列键。
- 列名含空格或筛选表达式含 shell 特殊字符时，请加引号。例如，`--filter '名称=*'` 匹配名称显示值为字面字符 `*` 的行。

```sh
ctyun ecs instance list --output json
ctyun ecs instance list --table compact
ctyun ecs instance list --table plain
ctyun ecs instance list --no-header
ctyun ecs instance list --filter 状态=running --sort "-实例 ID"
```

交互式表格会按终端显示宽度计算中文、英文、Emoji 等 Unicode 内容，并优先在空白或常见机器值分隔符处换行；输出重定向或通过管道传递时则保留自然宽度。`bordered`、`compact` 和 `plain` 样式共用同一套列宽计算和换行规则。

### 等待资源或任务状态

等待器需要核心版本 `>=0.5.0 <1.0.0`。在查询命令中使用 `--wait <waiter>`，轮询直到观察到文档定义的目标状态。命令帮助会列出适用的等待器，Shell 补全提供相同选项。发送请求前会检查等待器，拒绝不适用的命令和未声明安全轮询元数据的操作。轮询会重复执行查询命令，因此应使用之前变更操作返回的资源或任务标识。

```sh
ctyun ecs instance show {instance_id} --wait ecs.instance.running
ctyun vbs backup show --backup-id <backup_id> --wait vbs.backup.available
ctyun hpfs dataflow-task show --task-id <task_id> --wait hpfs.dataflow-task.completed
ctyun evs snapshot list --snapshot-id <snapshot_id> --wait evs.snapshot.available
ctyun ims image show {image_id} --wait ims.image.active
```

等待集合中的单个资源时，必须提供命令帮助要求的资源标识。每次响应都按该标识精确匹配：找不到资源时继续等待，匹配到多个资源时报告错误。

每个等待器通过 `max_attempts` 限制请求总数（含首次请求），通过 `interval_seconds` 设置两次请求之间的等待秒数。HTTP 超时对每次请求单独生效。在 64 位平台上，轮询间隔和单次请求超时的上限均为 9,223,372,036 秒。状态值为空或未知时会继续等待，直到达到轮询上限。等待器会输出失败或超时状态，命令退出码保持不变。JSON 模式下，首次响应写入标准输出，等待器状态写入标准错误。

## 核心更新

使用 `ctyun update --check` 检查核心更新，使用 `ctyun upgrade` 安装更新。指定 `--source auto` 时，CLI 先访问 GitHub；如果无法获取索引或签名，或者签名验证失败，则尝试 Gitee。CLI 下载所选索引引用的归档并验证其 SHA-256 校验和。可用 `--source github` 或 `--source gitee` 明确指定下载源。如果归档下载失败，可指定另一个源重新运行命令。通过 `--channel` 选择 `stable`、`beta` 或 `alpha` 通道。

在 Windows 上，升级成功后，`ctyun.exe` 同级目录下的 `.ctyun.exe.previous` 会保留上一版可执行文件。下次升级时，CLI 会先校验这份备份再替换它。如果旧进程仍在运行，请关闭后重试。如果备份目录内有无法识别的文件，请先将其移出再重试。

```sh
ctyun update --check --source auto
ctyun upgrade --source auto
ctyun upgrade --source auto --channel alpha
```

## 卸载

卸载核心二进制前，可先按需删除已安装插件和配置文件。删除插件会提示输入 `y/N` 确认；可按名称删除多个插件，也可删除全部插件。脚本中可使用 `--yes` 或 `-y` 跳过提示：

```sh
ctyun plugin list
ctyun plugin remove ecs region
ctyun plugin remove --all --yes
```

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

## 存储鉴权与文件传输

### 原生存储凭证

`media-storage` 和 `classic-object-storage` 插件要求核心版本 `>=0.5.0 <1.0.0`。其 `native` 命令使用 `CTYUN_STORAGE_ENDPOINT`（不包含桶名或路径的 HTTPS 服务端点）、`CTYUN_STORAGE_AK` 和 `CTYUN_STORAGE_SK`。`CTYUN_STORAGE_REGION` 应设置为服务文档中的签名区域，不能使用 OpenAPI 资源池 ID。`CTYUN_STORAGE_SIGNATURE_VERSION` 默认为 `v4`；使用 `v2` 时不需要签名区域。请求头签名使用 `CTYUN_STORAGE_SECURITY_TOKEN` 传递临时凭证，不复用 EOP 配置档的端点或凭证。媒体存储将桶名放在路径中，经典版将桶名放在主机名中。

经典版存储的 `native statistics` 使用 `s3` 签名服务和 `cn-mg` 等签名区域；`native tracking` 使用 `cloudtrail`，`native iam` 使用 `sts`，二者使用 `cn` 等签名区域。请根据对应服务文档选择端点和签名区域。操作跟踪和 IAM 要求使用 V4。IAM 的 `--tags` 接受由 `Key`/`Value` 对象组成的 JSON 数组，`--tag-keys` 接受 JSON 字符串数组。表单值由 CLI 执行 URL 编码，输入时无需预先编码。

### 策略上传

使用 `ctyun media-storage object post {bucket_name}` 时，提供 `--key` 和 `--file`。存储身份认证还需提供 `--storage-access-key` 和 Base64 编码的 JSON `--policy`，并通过 `--signature` 提供签名，或设置 `CTYUN_STORAGE_SK` 在本地计算 V2 签名。网关认证使用 `CTYUN_AK`/`CTYUN_SK`。原生 `object post` 使用相同的 V2 策略输入，路由只需 `CTYUN_STORAGE_ENDPOINT`；`CTYUN_STORAGE_SIGNATURE_VERSION` 不改变其策略算法。通过 `--security-token` 传递临时存储凭证。OpenAPI 网关使用 `success-action-status` 和 `success-action-redirect`，原生表单使用 `success_action_status` 和 `success_action_redirect`。成功重定向只返回目标地址，不会访问该地址。

### 文件输入与输出

仅在声明文件输入的命令上使用 `--file` 或 `--document-file`；普通 `@` 开头的值不会被当作文件读取。原生上传提供 `--content-type`，支持元数据的命令通过 `--metadata` 接收元数据名称与字符串值组成的 JSON 对象。上传需要临时磁盘空间。XML 输入文档和按显式传输策略解码的响应上限为 16 MiB。二进制下载以流式方式写入目标。命令帮助会列出可用输出格式：`--output raw` 原样输出响应体，不能与表格控制或等待器混用；下载命令另提供 `--output-file` 和 `--overwrite`。

### 分页与等待器

媒体存储原生列表的分页由用户显式控制：通过 `--output raw` 或 `--output json` 读取后续页标记，再传入下一次请求。对象可用等待器将有效的解冻对象和非归档对象视为可用；操作跟踪等待器检查跟踪是否开启或关闭，不代表日志投递正常。

## 开发者与贡献者工作流

开发和构建所需的最低 Go 版本为 1.26.0。

本项目默认跟随 [Go 官方支持窗口](https://go.dev/doc/devel/release#policy)，支持最新两个 Go 发布系列，通常以其中较旧系列的 `.0` 版本作为最低要求。语言特性、标准库 API、依赖或必要的编译器／运行时修复要求更高版本时，可以提高最低要求；不为兼容已停止支持的 Go 版本而长期保留过时依赖。`go.mod` 的 `go` 指令记录最低要求；开发和发布构建使用最新稳定版的补丁版本。本地工具链升级本身不要求同步提高最低版本或添加 `toolchain` 指令。

日常开发使用 Go 的共享构建缓存和模块缓存。覆盖率工具沿用其他 Go 命令的缓存设置。跨平台发布、反复运行覆盖率或竞态检测，或共享构建缓存只读时，可让同一批命令共用一个临时构建缓存。以下示例会在子 Shell 退出时清理缓存：

```sh
(
  build_cache=$(mktemp -d) || exit 1
  trap 'rm -rf "$build_cache"' EXIT
  export GOCACHE="$build_cache"
  go run ./tools/coverage
)
```

位置参数占位符使用花括号，例如 `{instance_id}`；选项值及 `<name>`、`<插件命令>` 等开发示例简写使用尖括号。运行示例前，请将占位符替换为实际值、命令或路径。

### 构建与测试

```sh
go run ./cmd/ctyun <插件命令> --offline
go run ./cmd/ctyun <插件命令> --fixture
go run ./cmd/ctyun --debug <插件命令> --offline
```

`--offline` 和 `--fixture` 都启用插件内置示例数据，不访问真实天翼云接口，适合本地调试命令形态、表格输出和参数映射。它们是仅供开发版使用的产品命令长选项，必须放在完整产品命令路径之后，不是全局选项，也没有短选项。发布版不会识别或暴露这些开发选项。

开发版可用 `--bundled` 从仓库内置插件元数据搜索、列出、安装、重新安装或更新插件。默认渠道为 `stable`。下方 ECS 示例选择其插件清单声明的 `beta` 渠道。开发版执行产品命令时也会优先使用仓库内置插件；这样即使同名发布插件已经安装，本地元数据改动也能直接验证。和 `--fixture` 一样，`--bundled` 面向开发和测试场景，不会出现在常规帮助中。

```sh
go run ./cmd/ctyun plugin list --available --channel all --bundled
go run ./cmd/ctyun plugin search ecs --channel beta --bundled
go run ./cmd/ctyun plugin install ecs --channel beta --bundled
go run ./cmd/ctyun plugin reinstall ecs --channel beta --bundled
go run ./cmd/ctyun plugin update ecs --channel beta --bundled
```

请使用普通用户权限运行测试。部分用例需要操作系统拒绝对只读文件或目录的写入。

```sh
git ls-files -co --exclude-standard '*.go' | sort -u | xargs gofmt -w
go vet ./...
go test ./internal/cli -run '^TestGoFilesStayUnderLineLimit$'
go test -timeout=30m ./...
go test ./internal/cli -run Completion -v
go test -timeout=30m ./tools/plugincheck
go run ./tools/coverage
```

覆盖率检查要求当前 Go 平台下未经筛选的 `.cache/coverage/raw.out` 达到 100% 语句覆盖率，直接检查执行次数，不排除任何文件或代码行，也不接受四舍五入后的达标结果。测试范围包括命令入口、维护工具、归档辅助代码和文件系统失败路径。

插件变更后建议按影响范围验证。先校验被修改的插件，再跑对应离线命令；如果改动影响通用插件加载、命令解析或表格输出，还应运行相应 Go 包的测试。

```sh
go run ./cmd/ctyun plugin lint ./plugins/<name>
go run ./cmd/ctyun <插件命令> --offline

go test -timeout=30m ./tools/plugincheck
go test ./internal/cli ./internal/plugin ./internal/output
```

### 验证安装脚本

在仓库根目录运行以下测试。可使用目标操作系统或对应的 CI 运行器，例如 GitHub Actions 的 `ubuntu-latest`、`macos-latest` 或 `windows-latest`。测试通过本地 HTTP 服务提供归档，并将文件安装到临时目录。

```sh
go test -count=1 -v -run '^(TestBashInstallerSelection|TestWindowsInstallerLiteralPaths)$' ./tools/release
```

在 macOS、Linux 或 WSL 上，确认 `TestBashInstallerSelection` 通过。该测试覆盖紧凑和缩进两种索引格式、通道选择和失败诊断。在 Windows 上，`TestWindowsInstallerLiteralPaths` 使用 Windows PowerShell 验证含空格和方括号的路径。

要用已发布的 Linux 二进制文件验证当前工作区的 Bash 脚本，可在 Bash 或 Zsh 中运行以下命令。本机的 Docker 引擎需使用 Linux 容器。基于 Alpine 的 Bash 镜像使用 BusyBox 的 `awk` 解析索引，无需额外安装 JSON 解析器。容器以普通用户身份运行，只读挂载安装脚本，默认安装目录随容器一起删除。

```sh
docker run --rm --user 65534:65534 -e HOME=/tmp -v "$PWD/install.sh:/work/install.sh:ro" \
  bash:5.2 bash -ec '
    bash /work/install.sh
    "$HOME/.local/bin/ctyun" --version
    "$HOME/.local/bin/ctyun" --help'
```

核对可执行文件输出的版本与安装成功提示是否一致。要单独验证某个发布源，可在容器镜像名称前加入 `-e CTYUN_INSTALL_SOURCE=github` 或 `-e CTYUN_INSTALL_SOURCE=gitee`。要在 macOS 或 Windows 上验证已发布的安装脚本，可将 `CTYUN_INSTALL_DIR` 指向新建的临时目录，运行“安装”一节中对应平台的命令，再用该目录中的可执行文件运行 `--version` 和 `--help`。

### 维护插件目录

OpenAPI 证据目录流水线是开发工具，不会暴露为用户命令，也不会进入核心或插件发布包。它从规范化 JSON 输入开始，并把上游证据保存在 `openapi-catalogs/<name>/source.json`：

```sh
go run ./tools/openapi harvest <name> --input path/to/normalized-source.json
go run ./tools/openapi diff <name>
go run ./tools/openapi normalize-labels <name>
go run ./tools/openapi generate <name>
go run ./tools/openapi review <name>
```

以下机器可读记录供开发者和 AI 维护插件时使用。[产品覆盖清单](openapi-catalogs/product-coverage.json) 将已纳入插件的命令与官方产品目录对照，记录跨产品引用、经过核对的等价接口、范围外接口及已确认的文档障碍。等价覆盖采用目标产品的服务地址和响应契约，不表示旧目录中的响应格式也受到支持。[官方 API 复核记录](openapi-catalogs/official-review.json)汇总了最近一次全插件检查、有证据支持的更新、保留的文档缺口和终端节点阻塞项。

产品级记录补充了 [EIP](openapi-catalogs/eip/coverage.json)、[PostgreSQL](openapi-catalogs/rds-postgresql/coverage.json)、[Redis](openapi-catalogs/redis/coverage.json) 和 [MongoDB](openapi-catalogs/mongodb/coverage.json) 的接口缺口、源示例修复或等待器限制。

更新插件时，依次完成四步：补充来源证据、生成并复核草稿、检查生命周期等待器、将通过复核的草稿纳入插件。

#### 1. 更新来源证据

在 `openapi-catalogs/<name>/` 中维护以下文件：

| 文件                                | 用途                                         | Git 管理方式         |
|-------------------------------------|----------------------------------------------|----------------------|
| `source.json`                       | 当前上游证据、API 范围及用于生成插件的元数据 | 跟踪                 |
| `baseline.json`                     | 最近一次接受的快照，用于校验已纳入插件的内容 | 跟踪；由提升操作更新 |
| `draft/`、`changes.md`、`review.md` | 生成的插件草稿、变更报告及复核结果           | 忽略；需要时重新生成 |

变更尚待复核时，较新的 `source.json` 可以与 `baseline.json` 不同。在提升前，现有插件的来源指纹和 API 范围仍以 `baseline.json` 为准。日常变更历史由 Git 保存。

更新 `source.json` 时：

- 用 `product.api_scope` 声明支持的 API URI 范围。生成、复核和提升都必须遵守该范围。
- 用 `request_example` 保存完整请求示例，用 `example` 保存单个参数值。上游未提供可用示例时，在复核中确认缺口，并记录 `example_unavailable`。
- 保留推荐替代 API 的上游证据。只有推荐、没有弃用或下线说明的内容，不能转成弃用警告。

#### 2. 生成并复核草稿

使用 `diff`、`generate` 和 `review` 重新生成本地输出。`normalize-labels` 可修复 `source.json` 中的共享技术词大小写和已审核短语；无法可靠修复的标签保持原样，必须处理后才能通过复核。

复核对象是可运行的插件草稿，而不只是来源目录。检查 API 契约、命令绑定、帮助、示例和响应夹具：

- 普通 JSON 夹具和 HTTP 响应夹具必须与已采集的来源证据一致。数据缺失或被改动会阻止复核通过。
- 表格的行选择器、列键、列取值路径和列弃用信息须与来源目录一致。复核时可以调整布局和默认显示的列。
- 示例必须比 Usage 行提供更多信息，例如具体参数、有意义的选项、结构化值或其他行为。只重复命令路径的示例，包括未解析的路径占位符形式，都会被拒绝。
- 检查草稿中的每条命令是否提供了所有支持语言的描述，改写机械拼接的英文描述。检查示例是否缺少必填选项、使用了未声明的选项，或提供了与参数类型不符的值。
- 只有推荐目标能解析到已跟踪且已纳入插件的 CLI 命令时，才生成推荐帮助；否则在 `source.json` 中保留未解析的证据。插件加载时，跨插件引用是软依赖；发布检查则要求目标精确匹配、未被弃用，并且不存在推荐循环。

如果 API 的响应契约明确，但没有可用的成功响应示例，用 `fixture_unavailable` 记录原因。保留命令、响应校验和原始证据，不编造成功夹具。该命令不能使用 `--offline` 或 `--fixture`，缺失的示例也不能作为等待器证据。下载接口应在响应分支声明文档规定的成功 `media_type`，避免把 HTTP 200 错误响应保存为文件。

#### 3. 检查生命周期等待器

每次复核时，检查异步操作是否有安全的查询命令和明确记载的终止状态。等待器只能绑定到非危险且明确允许重试的查询操作。状态证据或资源标识参数不足时记录缺口，不要猜测操作何时完成。

等待器定义及上游证据保存在目录的 `waiters` 映射中。草稿中的等待器定义必须与来源目录一致，提升会保留已接受的定义。修改定义后，应重新生成并复核草稿，再执行提升。使用真实插件包进行离线测试，检查响应结构、终止结果以及命令级帮助和补全。

<details>
<summary>等待器元数据参考</summary>

| 字段                               | 要求                                                                                     |
|------------------------------------|------------------------------------------------------------------------------------------|
| `commands`                         | 显式绑定到精确命令 ID                                                                    |
| `path` 或 `xml_path`               | JSON 状态路径，或精确选择一个标量元素的绝对 XML 命名空间路径；XML 路径不能使用集合选择器 |
| `success`、`failure`               | 文档规定的终止值；失败值为空表示未记录失败条件                                           |
| `success_values`、`failure_values` | 需要时补充其他有文档依据的终止值                                                         |
| `max_attempts`、`interval_seconds` | 正数尝试次数上限和轮询间隔（秒）                                                         |
| `evidence`                         | 状态定义的来源和含义                                                                     |

对于集合响应，添加 `selector`：`path` 定位集合，`key` 标识各行，`value` 引用现有的 `$arg.<name>` 或标量字符串/整数 `$param.<name>`。状态路径相对于选中的行。检查每条已采集记录，精确保留数值标识，并让空值或未知状态继续等待。没有可用的唯一资源标识参数时，记录阻塞原因。

</details>

#### 4. 将通过复核的草稿纳入插件

提升前，确认草稿已通过复核，且声明的质量（`generated`、`reviewed` 或 `curated`）准确反映插件的整理程度。

生成过程根据 `source.json` 计算 `source_fingerprint`。对于已有插件，保留已提升清单中的版本、通道、质量和核心兼容范围，并在需要时提高核心最低版本：

| 元数据功能 | 核心最低版本 |
|------------|--------------|
| 基础生成插件包 | `0.3.1` |
| 非字符串类型的请求体参数 | `0.4.0` |
| 等待器、显式传输策略、原生存储、文件下载或 XML 响应表格 | `0.5.0` |
| 独立的请求头、查询参数和请求体绑定 | `0.5.1` |

重新生成不能降低已有的兼容要求，也不能重置插件版本、通道或质量。提升操作会更新插件元数据，并将 `baseline.json` 推进到已接受的来源快照：

```sh
go run ./tools/openapi promote <name>
```

### 打包与发布

发布打包工具会生成核心二进制归档、`core-index.json`、`core-index.sig`、安装脚本、插件归档、`index.json` 和 `index.sig`。开发阶段可通过测试中的假 HTTP 源验证签名和下载逻辑；正式发布资产服务于上面的安装、核心更新和插件更新流程。

- 固定标签 `core` 和 `plugins` 是仓库仅有的两个 GitHub 发布页及构建产物根路径：`core` 存放核心安装与更新产物，`plugins` 存放插件安装与更新产物。
- 普通版本标签照常创建，但不为其创建单独的 GitHub 发布页，也不在其下上传构建产物。
- Gitee 的 `core` 发布页保持相同结构；受单个发布页附件数量限制，Gitee 的固定 `plugins` 发布页只存放 `index.json` 和 `index.sig`，每个插件归档则上传到其已有 `releases/plugins/<name>/<version>` 标签对应的不可变 Gitee 发布页。Gitee 签名索引使用这些归档的绝对下载 URL，仍然只下载用户选择的插件。
- 实际版本和通道分别由签名的 `core-index.json` 与 `index.json` 决定；版本级变更历史记录在根目录及各插件的规范变更日志中。
- 复用输出目录时，打包工具按核心发布通道分别更新各操作系统和架构组合。新版本只发布部分平台时，其余平台仍可安装原版本，直到各自的新构建发布。签名核心索引按语义版本从高到低排列，安装脚本据此选择适合当前平台的最新版本。两个版本若仅构建元数据不同，新发布的构建在其覆盖的平台上优先。
- 重新打包已列入索引的核心或插件版本时，会更新该版本的条目。尚未列入索引的核心旧版本可以补充没有更高版本构建的平台。打包较旧的插件版本时，该插件在同一通道已发布的更高版本仍然保留。其他通道继续使用各自的条目。
- 每个核心版本以及每个插件的各个版本都只属于一个通道。合并后的索引若将同一版本标签分配到多个通道，打包工具会报错。
- 需要保留的本地发布构建使用 `--out dist`。在该目录复用当前已发布的索引和插件归档，这样即使 Go 工具链更新，内容未变的插件也能保留原有的压缩字节和校验和。打包工具验证新的核心索引、GitHub 插件索引和 Gitee 插件索引的签名后，会自动清理文件名和 SHA-256 校验和均与上一批签名索引匹配的过期本地归档，并保留各平台、各通道仍需的归档。请先完成本次发布在 GitHub 和 Gitee 上的上传，再复用该目录打包下一次发布。
- 更新固定发布资产后，应保留当前签名索引引用的归档、索引签名以及核心安装脚本，并移除不再被索引引用的旧归档；仍在索引中提供的预发布通道归档应继续保留。

输出目录根部的插件索引和归档用于 GitHub；`gitee/index.json` 与 `gitee/index.sig` 用于覆盖 Gitee 固定 `plugins` 发布页上的同名文件。`gitee/releases.json` 是发布操作清单，记录每个归档的插件名、版本、通道、目标标签、校验和及预期下载 URL，不作为用户下载资产上传。发布者应先创建或更新清单列出的 Gitee 版本发布并上传对应归档，再更新固定 `plugins` 发布页的签名索引。

核心和插件版本必须遵循 Semantic Versioning 2.0.0。发布版本不要加 `v` 前缀。预发布版本使用 `0.1.0-alpha.1` 一类版本号和 `alpha`/`beta` 通道；稳定发布使用 `0.1.0` 一类版本号和 `stable` 通道。`internal/version/version.go` 中的默认值只用于未打包的开发构建，发布打包会覆盖实际版本和通道。

签名索引还包含由最终合并条目生成的 `versions` 映射：核心使用 `versions.<channel>`，插件使用 `versions.<name>.<channel>`，每个值均为对应通道的最高语义版本。README 动态徽章读取这些映射；CLI 仍通过原有数组选择下载产物。

开发和测试专用环境变量：

| 变量                        | 用途                                                     |
|-----------------------------|----------------------------------------------------------|
| `CTYUN_INSTALL_BASE_URL`    | 覆盖安装脚本读取的发布根地址，用于本地或临时发布资产验证 |
| `CTYUN_RELEASE_PRIVATE_KEY` | 发布打包工具签名索引使用的私钥                           |
| `CTYUN_RELEASE_PUBLIC_KEY`  | 开发构建或私有分发验证中用于核心更新和插件索引验签的公钥 |

本地演练时，可生成临时签名密钥，将当前平台的发布包写入新建的临时目录。打包工具会从私钥派生公钥，并将公钥嵌入二进制文件。为已有发布渠道打包时，请使用已安装客户端所信任公钥对应的发布私钥。

```sh
go run ./tools/release --generate-key
release_dir=$(mktemp -d) &&
  CTYUN_RELEASE_PRIVATE_KEY="<上一步输出的私钥>" \
  go run ./tools/release --version 0.5.2 --channel stable --out "$release_dir" --gitee-plugin-download-root "https://gitee.com/ArvinZJC/ctyun-cli/releases/download" --platform "$(go env GOOS)/$(go env GOARCH)"
```

正式发布时，GitHub 仍是源码和 CI 产物的权威来源，Gitee 作为同步镜像提供更稳的国内访问路径。已安装的 `ctyun` 使用签名公钥验证托管的核心及插件索引，并检查归档 SHA-256；引导安装脚本采用前文安装章节说明的信任方式。

## 友情链接

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli)：另一个非官方天翼云 CLI，使用 Python 编写，适合偏好 Python 生态的用户参考。
