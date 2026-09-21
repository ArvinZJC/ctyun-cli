# ctyun-cli

[![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fcore%2F*&label=release)](../../releases)
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

- **与官方 CLI 的关系：** 天翼云已于 2026 年 7 月 2 日发布官方 `ctyun-cli`。本项目是独立维护的非官方实现。

  天翼云官方 `ctyun-cli` 的公开入口是：[官方 CLI 文档](https://www.ctyun.cn/document/11095072)。截至目前，它没有独立的官方产品主页。官方工具命令名是 `ctyun-cli`，本项目命令名是 `ctyun`，两者不会发生二进制命名冲突，可以同时出现在同一个 shell 环境中。两者都使用 `CTYUN_AK` / `CTYUN_SK` 作为 AK/SK 环境变量；如果同时使用，请留意这组凭据会被两个工具共享。

  天翼云官方 Go SDK 名为 `ctyun-go-sdk`，但产品覆盖有限，且未公开发布；如需官方 SDK，可向天翼云提交工单获取。本项目不是 SDK，而是面向用户操作流程的命令行工具。

  本项目会继续迭代，作为官方 CLI 之外的非官方选择。我们会继续探索和实现现代云 CLI 需要的能力，并把更好的终端体验、脚本友好性、可组合输出和可维护扩展方式放在重要位置；当官方 CLI 在稳定性、灵活性、能力深度和使用体验上足够成熟后，再重新评估本项目的定位与生命周期。

- 开通要管理的服务，并准备具有相应权限的 AK/SK 凭据。

- 本工具支持自研池（一类节点）的客户侧（C 端）接口，不支持业务／运营侧（B 端）接口和合营池（二类节点）。

- 服务的具体使用条件请参阅[天翼云 OpenAPI 文档](https://eop.ctyun.cn/ebp/ctapiDocument/index)。

## 安装

可通过安装脚本安装原生 `ctyun` 二进制。默认脚本按 `stable`、`beta`、`alpha` 的顺序选择第一个可用通道；如需固定通道，可设置 `CTYUN_INSTALL_CHANNEL`。如果 GitHub 访问不稳定，可把 URL 中的 `github.com` 替换为 `gitee.com`。

macOS、Linux 和 WSL：

```sh
curl -fsSL https://github.com/ArvinZJC/ctyun-cli/releases/download/core/install.sh | bash
```

Windows PowerShell：

```powershell
irm https://github.com/ArvinZJC/ctyun-cli/releases/download/core/install.ps1 | iex
```

如果不确定当前终端是否为 PowerShell，请先从开始菜单或 Windows Terminal 的标签页菜单打开 Windows PowerShell，再运行 `$PSVersionTable.PSVersion` 确认；能看到版本信息后，在同一个窗口运行上面的安装命令。

引导安装脚本从选定的发布源下载索引和归档（默认源使用 HTTPS），并根据索引中的 SHA-256 校验归档。脚本不验证 `core-index.sig`，因此首次安装依赖所下载脚本及发布源的可信性。安装完成后，`ctyun` 在核心更新和托管插件操作中同时验证索引签名与归档哈希。

安装脚本支持这些环境变量：

| 变量                    | 用途                                                                                                         |
|-------------------------|--------------------------------------------------------------------------------------------------------------|
| `CTYUN_INSTALL_CHANNEL` | 固定安装通道，可设为 `stable`、`beta` 或 `alpha`                                                             |
| `CTYUN_INSTALL_SOURCE`  | 固定安装源，可设为 `auto`、`github` 或 `gitee`                                                               |
| `CTYUN_INSTALL_DIR`     | 覆盖安装目录；默认 macOS、Linux 和 WSL 为 `$HOME/.local/bin`，Windows 为 `%LOCALAPPDATA%\Programs\ctyun-cli` |

## 鉴权、配置与语言

实时请求优先从进程环境读取 AK/SK：

```sh
export CTYUN_AK=...
export CTYUN_SK=...
```

如果 `CTYUN_AK` 或 `CTYUN_SK` 缺失，`ctyun` 会按当前配置档案、全局配置的顺序读取 `ak`/`sk`。当实时命令实际使用配置中的 AK/SK 时，会向标准错误输出（stderr）写入提醒；可设置环境变量 `CTYUN_WARN_CONFIG_CREDENTIALS=0`，或运行 `ctyun config set warn_config_credentials false` 关闭。

### 配置与配置档案

配置文件按 `--config`、`CTYUN_CONFIG`、`~/.ctyun/config.json` 的顺序选择；`--profile` 覆盖 `active_profile`。其他设置依次采用命令行选项、环境变量、当前配置档案和支持的全局后备值。

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

`ctyun config reset` 会先提示确认；确认后创建备份，再删除当前配置文件。脚本中可使用 `--yes` 或 `-y` 跳过提示。

### 语言

支持的语言为 `zh-CN`、`en-US` 和 `en-GB`。语言选择顺序为 `--lang`、`CTYUN_LANGUAGE`、配置档案中的 `language`、系统语言；无法匹配时默认 `zh-CN`。

## 插件

新安装的 `ctyun` 只包含核心命令，不会预装产品插件。产品命令来自插件包；完成鉴权、配置和语言设置后，请先安装所需插件：

```sh
ctyun plugin search ecs --source auto
ctyun plugin list --available --source auto
ctyun plugin list --available --cols 插件,质量,状态 --filter 状态=可安装 --source auto
ctyun plugin install region --source auto
ctyun plugin install ecs --source auto --channel beta
ctyun plugin install --all --source auto
ctyun plugin list
```

<details>
<summary>插件列表</summary>

| 名称                     | 插件                     | 产品                     | 版本                                                                                                                                                         | 通道     | 质量        | 命令 | 操作 |
|--------------------------|--------------------------|--------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|-------------|-----:|-----:|
| 应用云主机 ACS           | `acs`                    | `acs`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Facs%2F*&label=release)](../../releases)                    | `beta`   | `generated` |   15 |   15 |
| 弹性伸缩服务 AS          | `as`                     | `as`                     | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fas%2F*&label=release)](../../releases)                     | `beta`   | `generated` |   63 |   63 |
| 云备份 CBR               | `cbr`                    | `cbr`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcbr%2F*&label=release)](../../releases)                    | `beta`   | `generated` |   22 |   22 |
| 云容灾 CDR               | `cdr`                    | `cdr`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcdr%2F*&label=release)](../../releases)                    | `beta`   | `generated` |   31 |   31 |
| 函数计算                 | `cf`                     | `cf`                     | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcf%2F*&label=release)](../../releases)                     | `beta`   | `generated` |   62 |   62 |
| 云助手                   | `cloud-assistant`        | `cloud-assistant`        | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcloud-assistant%2F*&label=release)](../../releases)        | `beta`   | `generated` |   11 |   11 |
| 公共服务                 | `common`                 | `common`                 | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fcommon%2F*&label=release)](../../releases)                 | `stable` | `curated`   |    1 |    1 |
| 物理机 DPS               | `dps`                    | `dps`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fdps%2F*&label=release)](../../releases)                    | `beta`   | `generated` |   59 |   59 |
| 天翼云电脑（政企版）     | `ecpc`                   | `ecpc`                   | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fecpc%2F*&label=release)](../../releases)                   | `beta`   | `generated` |  303 |  303 |
| 弹性云主机 ECS           | `ecs`                    | `ecs`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fecs%2F*&label=release)](../../releases)                    | `beta`   | `generated` |  209 |  209 |
| 弹性高性能计算 E-HPC     | `ehpc`                   | `ehpc`                   | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fehpc%2F*&label=release)](../../releases)                   | `beta`   | `generated` |   24 |   24 |
| 弹性IP EIP               | `eip`                    | `eip`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Feip%2F*&label=release)](../../releases)                    | `beta`   | `generated` |   57 |   57 |
| 弹性负载均衡 ELB         | `elb`                    | `elb`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Felb%2F*&label=release)](../../releases)                    | `beta`   | `generated` |  114 |  114 |
| 云硬盘 EVS               | `evs`                    | `evs`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fevs%2F*&label=release)](../../releases)                    | `beta`   | `generated` |   36 |   36 |
| 高性能并行文件服务 HPFS  | `hpfs`                   | `hpfs`                   | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fhpfs%2F*&label=release)](../../releases)                   | `beta`   | `generated` |   40 |   40 |
| 镜像服务 IMS             | `ims`                    | `ims`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fims%2F*&label=release)](../../releases)                    | `beta`   | `generated` |   27 |   27 |
| 任务                     | `job`                    | `job`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fjob%2F*&label=release)](../../releases)                    | `stable` | `curated`   |    1 |    1 |
| NAT网关                  | `nat`                    | `nat`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fnat%2F*&label=release)](../../releases)                    | `beta`   | `generated` |   22 |   22 |
| 海量文件服务 OceanFS     | `oceanfs`                | `oceanfs`                | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Foceanfs%2F*&label=release)](../../releases)                | `beta`   | `generated` |   34 |   34 |
| 对象存储（经典版）I型    | `classic-object-storage` | `classic-object-storage` | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fclassic-object-storage%2F*&label=release)](../../releases) | `beta`   | `generated` |  219 |  219 |
| 媒体存储                 | `media-storage`          | `media-storage`          | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fmedia-storage%2F*&label=release)](../../releases)          | `beta`   | `generated` |  124 |  124 |
| 订单                     | `order`                  | `order`                  | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Forder%2F*&label=release)](../../releases)                  | `stable` | `curated`   |    7 |    7 |
| 私网NAT网关              | `private-nat`            | `private-nat`            | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fprivate-nat%2F*&label=release)](../../releases)            | `beta`   | `generated` |   21 |   21 |
| 关系数据库MySQL版        | `rds-mysql`              | `rds-mysql`              | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Frds-mysql%2F*&label=release)](../../releases)              | `beta`   | `generated` |  224 |  224 |
| 关系数据库 PostgreSQL 版 | `rds-postgresql`         | `rds-postgresql`         | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Frds-postgresql%2F*&label=release)](../../releases)         | `beta`   | `generated` |  153 |  153 |
| 关系型数据库 SQL Server  | `rds-sqlserver`          | `rds-sqlserver`          | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Frds-sqlserver%2F*&label=release)](../../releases)          | `beta`   | `generated` |  114 |  114 |
| 资源池                   | `region`                 | `region`                 | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fregion%2F*&label=release)](../../releases)                 | `stable` | `curated`   |    7 |    7 |
| 弹性文件服务 SFS         | `sfs`                    | `sfs`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fsfs%2F*&label=release)](../../releases)                    | `beta`   | `generated` |   56 |   56 |
| 云硬盘备份 VBS           | `vbs`                    | `vbs`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fvbs%2F*&label=release)](../../releases)                    | `beta`   | `generated` |   37 |   37 |
| 虚拟私有云 VPC           | `vpc`                    | `vpc`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fvpc%2F*&label=release)](../../releases)                    | `beta`   | `generated` |  193 |  193 |
| 对象存储 ZOS             | `zos`                    | `zos`                    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fzos%2F*&label=release)](../../releases)                    | `beta`   | `generated` |  106 |  106 |

质量字段表示插件元数据的整理程度：`generated` 表示工具生成的初稿，`reviewed` 表示已完成基础复核，`curated` 表示作为维护版本持续更新。

</details>

使用以下插件时，请留意各自的适用范围、配置要求和功能限制：

- 网络：`nat` 管理公网 NAT 网关，`private-nat` 管理私网 NAT 网关。暂不支持旧版 EIP 网络查询，详见[覆盖清单](openapi-catalogs/eip/coverage.json)。
- RDS：暂不支持 6 个 PostgreSQL 下载或导出接口，详见 [覆盖清单](openapi-catalogs/rds-postgresql/coverage.json)。已下线的 MySQL 跨地域备份目标地域查询命令仍保留弃用警告，但无法在线调用。
- 存储：`media-storage` 和 `classic-object-storage` 的 `native` 命令使用独立的存储凭证，详见[存储鉴权与文件传输](#存储鉴权与文件传输)。媒体存储 OpenAPI 网关适用于西藏资源池 1 区。
- 云助手：上游 API 暂时下线，`ctyun cloud-assistant command run` 目前无法在线调用。命令仍予以保留，此次暂停不代表永久弃用。

插件管理命令共享这些行为：

- `ctyun plugin search`、`ctyun plugin list --available`、`ctyun plugin install`、`ctyun plugin reinstall` 和 `ctyun plugin update` 都支持 `--source` 和 `--channel`。
- `ctyun plugin list --available` 会显示托管插件及本地安装状态。
- `ctyun plugin list --available` 和 `ctyun plugin search` 默认查看 `stable` 通道，也可使用 `--channel all` 查看所有插件源通道。
- 安装、重装、更新和更新检查默认选择 `stable` 通道；如需选择预发布插件，请显式指定 `--channel beta` 或 `--channel alpha`。
- `ctyun plugin search` 支持模糊搜索，并遵循表格/JSON 输出控制。
- `ctyun plugin install` 只安装尚未安装的插件；如果插件已安装，则跳过，不会通过 install 升级、降级或覆盖现有版本。
- `ctyun plugin reinstall` 只处理已安装插件，并会按指定源刷新插件；重装允许覆盖相同版本，也允许显式切换到所选通道中的较低版本。
- `ctyun plugin update` 只安装 SemVer 优先级更高的版本。
- 安装、重装、更新、删除和核心升级在交互式终端中通过 stderr 显示进度，完成后只向 stdout 输出一条汇总；重定向或管道场景不会输出进度控制字符。

```sh
ctyun plugin reinstall region --source auto
ctyun plugin reinstall ecs --source auto --channel beta
ctyun plugin reinstall --all --source auto
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

- `--cols`、`--filter` 和 `--sort` 可使用表格中看到的列名，也兼容稳定列键。
- 只有当参数值会被 shell 拆开时才需要加引号，例如使用带空格的英文列名。

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

每个等待器定义 `max_attempts`（包含首次请求）和 `interval_seconds`，两者与 HTTP 请求超时独立。空值和未知值保持等待，直到达到轮询上限。失败和超时会作为最终等待器状态输出，目前不会改变命令退出状态。JSON 模式仍将首次响应写入标准输出，将等待器状态写入标准错误。

## 核心更新

使用 `ctyun update` 或 `ctyun upgrade` 检查并安装核心更新。通过 `--source` 选择 `auto`、`github` 或 `gitee`；`auto` 先尝试 GitHub，失败后回退到 Gitee。更新时会验证索引签名和归档 SHA-256。通过 `--channel` 选择 `stable`、`beta` 或 `alpha` 通道。

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

如需清理配置文件，可运行：

```sh
ctyun config reset
```

macOS、Linux 和 WSL 可用 `command -v` 定位当前 `PATH` 上的 `ctyun` 后删除；默认安装路径是 `$HOME/.local/bin/ctyun`：

```sh
ctyun_path="$(command -v ctyun)" && rm -f "$ctyun_path"
```

Windows PowerShell 默认安装到 `%LOCALAPPDATA%\Programs\ctyun-cli\ctyun.exe`；如果安装时设置过 `CTYUN_INSTALL_DIR`，请使用同一个目录：

```powershell
$InstallDir = if ($env:CTYUN_INSTALL_DIR) { $env:CTYUN_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA "Programs\ctyun-cli" }
Remove-Item -Force (Join-Path $InstallDir "ctyun.exe") -ErrorAction SilentlyContinue
```

## 存储鉴权与文件传输

### 原生存储凭证

`media-storage` 和 `classic-object-storage` 插件要求核心版本 `>=0.5.0 <1.0.0`。其 `native` 命令使用 `CTYUN_STORAGE_ENDPOINT`（不包含桶名或路径的 HTTPS 服务端点）、`CTYUN_STORAGE_AK` 和 `CTYUN_STORAGE_SK`。`CTYUN_STORAGE_REGION` 应设置为服务文档中的签名区域，不能使用 OpenAPI 资源池 ID。`CTYUN_STORAGE_SIGNATURE_VERSION` 默认为 `v4`；使用 `v2` 时不需要签名区域。请求头签名使用 `CTYUN_STORAGE_SECURITY_TOKEN` 传递临时凭证，不复用 EOP 配置档的端点或凭证。媒体存储将桶名放在路径中，经典版将桶名放在主机名中。

经典版存储的 `native statistics` 使用 `s3` 签名服务和 `cn-mg` 等签名区域；`native tracking` 使用 `cloudtrail`，`native iam` 使用 `sts`，二者使用 `cn` 等签名区域。请根据对应服务文档选择端点和签名区域。操作跟踪和 IAM 要求使用 V4。IAM 的 `--tags` 接受由 `Key`/`Value` 对象组成的 JSON 数组，`--tag-keys` 接受 JSON 字符串数组。表单值由 CLI 执行 URL 编码，输入时无需预先编码。

### 策略上传

使用 `ctyun media-storage object post {bucket_name}` 时，提供 `--key` 和 `--file`。存储身份认证还需提供 `--storage-access-key` 和 Base64 编码的 JSON `--policy`，并通过 `--signature` 提供签名，或设置 `CTYUN_STORAGE_SK` 在本地计算 V2 签名。网关认证使用 `CTYUN_AK`/`CTYUN_SK`。原生 `object post` 使用相同的 V2 策略输入，路由只需 `CTYUN_STORAGE_ENDPOINT`；`CTYUN_STORAGE_SIGNATURE_VERSION` 不改变其策略算法。通过 `--security-token` 传递临时存储凭证。OpenAPI 网关使用 `success-action-status` 和 `success-action-redirect`，原生表单使用 `success_action_status` 和 `success_action_redirect`。成功重定向只返回目标地址，不会访问该地址。

### 文件输入与输出

仅在声明文件输入的命令上使用 `--file` 或 `--document-file`；普通 `@` 开头的值不会被当作文件读取。原生上传提供 `--content-type`，支持元数据的命令通过 `--metadata` 接收元数据名称与字符串值组成的 JSON 对象。上传需要临时磁盘空间，结构化响应和 XML 文档限为 16 MiB。命令帮助会列出可用输出格式：`--output raw` 原样输出响应体，不能与表格控制或等待器混用；下载命令另提供 `--output-file` 和 `--overwrite`。

### 分页与等待器

媒体存储原生列表的分页由用户显式控制：通过 `--output raw` 或 `--output json` 读取后续页标记，再传入下一次请求。对象可用等待器将有效的解冻对象和非归档对象视为可用；操作跟踪等待器检查跟踪是否开启或关闭，不代表日志投递正常。

## 开发者与贡献者工作流

开发和构建需要 Go 1.26.0 或更高版本，以满足当前依赖的最低要求。

本项目默认跟随 [Go 官方支持窗口](https://go.dev/doc/devel/release#policy)，支持最新两个 Go 发布系列，通常以其中较旧系列的 `.0` 版本作为最低要求。语言特性、标准库 API、依赖或必要的编译器／运行时修复要求更高版本时，可以提高最低要求；不为兼容已停止支持的 Go 版本而长期保留过时依赖。`go.mod` 的 `go` 指令记录最低要求；开发和发布构建使用最新稳定版的补丁版本。本地工具链升级本身不要求同步提高最低版本或添加 `toolchain` 指令。

如果默认 Go 构建缓存不可写（例如在沙盒环境中），先设置仓库内缓存：

```sh
export GOCACHE="$PWD/.cache/go-build"
```

位置参数占位符使用花括号，例如 `{instance_id}`；选项值及 `<name>`、`<插件命令>` 等开发示例简写使用尖括号。运行示例前，请将占位符替换为实际值、命令或路径。

### 构建与测试

```sh
go run ./cmd/ctyun <插件命令> --offline
go run ./cmd/ctyun <插件命令> --fixture
go run ./cmd/ctyun --debug <插件命令> --offline
```

`--offline` 和 `--fixture` 都启用插件内置示例数据，不访问真实天翼云接口，适合本地调试命令形态、表格输出和参数映射。它们是仅供开发版使用的产品命令长选项，必须放在完整产品命令路径之后，不是全局选项，也没有短选项。发布版不会识别或暴露这些开发选项。

开发版可用 `--bundled` 从仓库内置插件元数据搜索、列出、安装、重新安装或更新插件。开发版执行产品命令时也会优先使用仓库内置插件；这样即使同名发布插件已经安装，本地元数据改动也能直接验证。和 `--fixture` 一样，`--bundled` 面向开发和测试场景，不会出现在常规帮助中。

```sh
go run ./cmd/ctyun plugin list --available --bundled
go run ./cmd/ctyun plugin search <name> --bundled
go run ./cmd/ctyun plugin install <name> --bundled
go run ./cmd/ctyun plugin reinstall <name> --bundled
go run ./cmd/ctyun plugin update <name> --bundled
```

测试：

```sh
git ls-files -co --exclude-standard '*.go' | sort -u | xargs gofmt -w
go vet ./...
go test ./internal/cli -run '^TestGoFilesStayUnderLineLimit$'
go test ./...
go test ./internal/cli -run Completion -v
go test ./tools/plugincheck
go run ./tools/coverage
```

插件变更后建议按影响范围验证。先校验被修改的插件，再跑对应离线命令；如果改动影响通用插件加载、命令解析或表格输出，再补充相关 Go 测试。

```sh
go run ./cmd/ctyun plugin lint ./plugins/<name>
go run ./cmd/ctyun <插件命令> --offline

go test ./tools/plugincheck
go test ./internal/cli ./internal/plugin ./internal/output
```

### 维护插件目录

OpenAPI 证据目录流水线是开发工具，不会暴露为用户命令，也不会进入核心或插件发布包。它从规范化 JSON 输入开始，并把上游证据保存在 `openapi-catalogs/<name>/source.json`：

```sh
go run ./tools/openapi harvest <name> --input path/to/normalized-source.json
go run ./tools/openapi diff <name>
go run ./tools/openapi normalize-labels <name>
go run ./tools/openapi generate <name>
go run ./tools/openapi review <name>
```

对通过该流水线维护的插件：

- 跟踪对应的 `source.json` 作为上游证据，并跟踪提升后更新的 `baseline.json` 作为最近一次接受的快照。上游证据更新后，在完成复核和提升前，`source.json` 与已提升插件或 `baseline.json` 存在差异是预期状态；已提升插件的来源指纹和 API 范围仍以 `baseline.json` 为准。
- 每次插件评审都应检查生命周期等待器。目录中的 `waiters` 将等待器 ID 映射到 `commands`（精确命令 ID）、状态 `path`（JSON）或 `xml_path`（XML 命名空间与元素名组成的绝对路径，精确匹配一个标量元素，不能与集合选择器混用）、单值 `success`/`failure`、可选的额外 `success_values`/`failure_values`、正数 `max_attempts`/`interval_seconds`，以及注明文档状态语义的 `evidence`。空的单值失败条件表示上游未提供失败状态。显式绑定必须指向非危险且可重试的查询操作。评审拒绝草稿等待器漂移；提升会保留定义并推进基线。增加离线插件检查，覆盖响应结构、终止状态和命令级帮助/补全；对于集合响应，添加 `selector`，其中 `path` 指向集合、`key` 指向行内标识、`value` 引用现有的 `$arg.<name>` 或字符串/整数 `$param.<name>`；状态路径相对于匹配行。检查每条已采集记录的结构，精确保留数值标识，空值或未知状态保持等待。状态证据或唯一标识输入不足时记录缺口。
- 已发布的接口若有明确的响应契约，但没有可用的成功响应示例，使用 `fixture_unavailable` 记录原因。下载接口若明确规定成功响应的媒体类型，应在响应分支中声明 `media_type`，避免将 HTTP 200 错误响应保存为文件。保留命令和响应校验，不生成成功响应样例，并在目录中保留原始响应依据。此类命令不能使用 `--offline` 或 `--fixture`，也不能提供等待器所需的响应证据。
- 用 `product.api_scope` 记录该插件覆盖的上游 API URI 范围；生成、复核和提升时不要把范围外的 API 静默纳入插件。
- 对只有推荐、没有弃用或下线说明的上游内容，在 `source.json` 中保留目标 API 证据；如果尚不能解析到已跟踪且已提升的可见命令，就保持未解析状态，不生成命令帮助元数据。插件加载时，跨插件命令引用保持软依赖；引用一旦进入仓库中已提升的插件元数据，发布检查必须确认它精确解析到未弃用的目标命令，并拒绝推荐循环。
- 在 `source.json` 中保留可执行示例所需的上游证据：完整请求使用 `request_example`，单个参数值使用 `example`；上游确实没有可用值时，复核后明确记录 `example_unavailable`。只重复 Usage 已展示命令路径的示例（包括未解析的路径占位符形式）不会生成，仓库发布检查也会拒绝这类冗余示例；示例应提供具体参数、有意义的选项、结构化值或其他额外行为。复核还会拒绝机械拼接的英文描述、缺少必填命令选项的示例、未声明的选项以及与参数类型不匹配的值。
- `normalize-labels` 只对 `source.json` 执行共享技术词大小写和已审核短语的保守修复；无法可靠修复的标签保持原样，并继续阻止复核通过。
- `draft/`、`changes.md` 和 `review.md` 是可复现的本地复核输出，默认忽略；需要复核时重新运行 `diff`、`generate` 和 `review`。
- 生成草稿会从 `source.json` 写入 `source_fingerprint`；已有插件的版本、通道、质量和核心兼容范围沿用已提升清单；目录声明等待器时，将核心最低版本提升到 0.5.0，避免重新生成时降级发布身份。草稿通过复核、且 `generated`/`reviewed`/`curated` 质量值准确反映当前整理程度时，运行提升命令会更新插件元数据并推进 `baseline.json`。
- 普通历史由 git 保存。

```sh
go run ./tools/openapi promote <name>
```

### 打包与发布

发布打包工具会生成核心二进制归档、`core-index.json`、`core-index.sig`、安装脚本、插件归档、`index.json` 和 `index.sig`。开发阶段可通过测试中的假 HTTP 源验证签名和下载逻辑；正式发布资产服务于上面的安装、核心更新和插件更新流程。

- 固定标签 `core` 和 `plugins` 是仓库仅有的两个 GitHub 发布页及构建产物根路径：`core` 存放核心安装与更新产物，`plugins` 存放插件安装与更新产物。
- 普通版本标签照常创建，但不为其创建单独的 GitHub 发布页，也不在其下上传构建产物。
- Gitee 的 `core` 发布页保持相同结构；受单个发布页附件数量限制，Gitee 的固定 `plugins` 发布页只存放 `index.json` 和 `index.sig`，每个插件归档则上传到其已有 `releases/plugins/<name>/<version>` 标签对应的不可变 Gitee 发布页。Gitee 签名索引使用这些归档的绝对下载 URL，仍然只下载用户选择的插件。
- 实际版本和通道分别由签名的 `core-index.json` 与 `index.json` 决定；版本级变更历史记录在根目录及各插件的规范变更日志中。
- 对已有输出目录再次运行打包工具时，它会保留其他通道的现有索引条目，只替换本次重新构建的核心通道或插件名/通道资产，然后重新签名索引；如果为同一核心版本补充平台归档，则会合并平台资产。
- 更新固定发布资产后，应保留当前签名索引引用的归档、索引签名以及核心安装脚本，并移除不再被索引引用的旧归档；仍在索引中提供的预发布通道归档应继续保留。

输出目录根部的插件索引和归档用于 GitHub；`gitee/index.json` 与 `gitee/index.sig` 用于覆盖 Gitee 固定 `plugins` 发布页上的同名文件。`gitee/releases.json` 是发布操作清单，记录每个归档的插件名、版本、通道、目标标签、校验和及预期下载 URL，不作为用户下载资产上传。发布者应先创建或更新清单列出的 Gitee 版本发布并上传对应归档，再更新固定 `plugins` 发布页的签名索引。

核心和插件版本必须遵循 Semantic Versioning 2.0.0。发布版本不要加 `v` 前缀。预发布版本使用 `0.1.0-alpha.1` 一类版本号和 `alpha`/`beta` 通道；稳定发布使用 `0.1.0` 一类版本号和 `stable` 通道。`internal/version/version.go` 中的默认值只用于未打包的开发构建，发布打包会覆盖实际版本和通道。

开发和测试专用环境变量：

| 变量                        | 用途                                                     |
|-----------------------------|----------------------------------------------------------|
| `CTYUN_INSTALL_BASE_URL`    | 覆盖安装脚本读取的发布根地址，用于本地或临时发布资产验证 |
| `CTYUN_RELEASE_PRIVATE_KEY` | 发布打包工具签名索引使用的私钥                           |
| `CTYUN_RELEASE_PUBLIC_KEY`  | 开发构建或私有分发验证中用于核心更新和插件索引验签的公钥 |

```sh
go run ./tools/release --generate-key
export CTYUN_RELEASE_PRIVATE_KEY="<上一步输出的私钥>"
export CTYUN_RELEASE_PUBLIC_KEY="<上一步输出的公钥>"
go run ./tools/release --version 0.5.0 --channel stable --out ./dist/releases --gitee-plugin-download-root "https://gitee.com/ArvinZJC/ctyun-cli/releases/download" --platform "$(go env GOOS)/$(go env GOARCH)"
```

正式发布时，GitHub 仍是源码和 CI 产物的权威来源，Gitee 作为同步镜像提供更稳的国内访问路径。已安装的 `ctyun` 使用签名公钥验证托管的核心及插件索引，并检查归档 SHA-256；引导安装脚本采用前文安装章节说明的信任方式。

## 友情链接

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli)：另一个非官方天翼云 CLI，使用 Python 编写，适合偏好 Python 生态的用户参考。
