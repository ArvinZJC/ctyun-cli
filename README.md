# ctyun-cli

[![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fcore%2F*&label=release)](../../releases)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/ArvinZJC/ctyun-cli)
[![GitHub License](https://img.shields.io/github/license/ArvinZJC/ctyun-cli?label=licence)](./LICENCE)

简体中文 | [English](./README-EN.md)

`ctyun-cli` 是仓库名，`ctyun` 是命令行工具名。这是一个使用 Go 编写、基于天翼云 OpenAPI 的非官方 CLI：插件化，体验优先，面向终端里的天翼云资源查询和管理。（目前天翼云没有官方 CLI。）

天翼云官方 Go SDK 名为 `ctyun-go-sdk`，但产品覆盖有限，且未公开发布；如需官方 SDK，可向天翼云提交工单获取。本项目不是 SDK，而是面向用户操作流程的命令行工具。

## 使用前须知

- 请先开通要操作的天翼云服务，并了解对应 OpenAPI 的基本用法。
- 本项目基于天翼云 OpenAPI 的 C 端接口，即消费者/客户侧接口。
- 不支持 B 端接口，即业务/运营侧接口。
- 支持一类节点，即自研池；不支持二类节点，即合营池。
- 仅支持 AK/SK 鉴权，因为天翼云 OpenAPI 当前只支持 AK/SK。

OpenAPI 文档入口：[天翼云 OpenAPI 文档](https://eop.ctyun.cn/ebp/ctapiDocument/index)。其中的 API 文档就是这里提到的 C 端接口文档。

## 亮点

- 默认表格输出，适合人工查看，支持中英文内容宽度对齐。
- 支持 `--output json`，便于脚本和其他工具处理。
- 产品命令由插件元数据提供，核心 CLI 不需要为每个产品写专门的分支。
- 插件可声明请求方法、路径、参数、表格列、示例、等待器和危险操作确认。
- 支持国际化：核心帮助、错误提示、运行时提醒、插件名称、命令说明和表格列都可以本地化。

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

安装脚本支持这些环境变量：

| 变量                      | 用途                                                                                             |
|-------------------------|------------------------------------------------------------------------------------------------|
| `CTYUN_INSTALL_CHANNEL` | 固定安装通道，可设为 `stable`、`beta` 或 `alpha`                                                           |
| `CTYUN_INSTALL_SOURCE`  | 固定安装源，可设为 `auto`、`github` 或 `gitee`                                                            |
| `CTYUN_INSTALL_DIR`     | 覆盖安装目录；默认 macOS、Linux 和 WSL 为 `$HOME/.local/bin`，Windows 为 `%LOCALAPPDATA%\Programs\ctyun-cli` |

## 核心命令

这些命令不依赖产品插件，适合安装后先确认版本、查看帮助、生成补全脚本或检查网络连通性：

```sh
ctyun --version
ctyun help
ctyun help config
ctyun completion zsh
ctyun doctor network
```

插件命令的帮助会在安装对应插件后可用，例如 `ctyun help region list`。

## 鉴权、配置与语言

配置文件查找顺序为 `--config`、`CTYUN_CONFIG`、`~/.ctyun/config.json`；`--profile` 会覆盖 `active_profile`。除这类用于定位配置文件的选项外，运行时设置遵循“命令行选项、环境变量、当前配置档案、支持的全局配置后备”的顺序；同一设置同时出现在环境变量和配置中时，环境变量优先。`CTYUN_CONFIG` 是例外：它用于找到配置文件，因此不会再从配置文件读取自身的后备值。

常用环境变量：

| 变量                              | 用途                                            |
|---------------------------------|-----------------------------------------------|
| `CTYUN_CONFIG`                  | 覆盖配置文件路径                                      |
| `CTYUN_AK`                      | 实时请求使用的天翼云 AK                                 |
| `CTYUN_SK`                      | 实时请求使用的天翼云 SK                                 |
| `CTYUN_LANGUAGE`                | 覆盖界面语言，可设为 `zh-CN`、`en-US` 或 `en-GB`          |
| `CTYUN_WARN_CONFIG_CREDENTIALS` | 设为 `0` 可关闭使用配置中 AK/SK 时的提醒                    |
| `CTYUN_PLUGIN_SOURCE`           | 插件安装、搜索和更新的默认来源，可设为 `auto`、`github` 或 `gitee` |
| `CTYUN_UPGRADE_SOURCE`          | 核心更新的默认来源，可设为 `auto`、`github` 或 `gitee`       |

实时请求优先从进程环境读取 AK/SK：

```sh
export CTYUN_AK=...
export CTYUN_SK=...
```

如果 `CTYUN_AK` 或 `CTYUN_SK` 缺失，`ctyun` 会按当前配置档案、全局配置的顺序读取 `ak`/`sk`。当实时命令实际使用配置中的 AK/SK 时，会向标准错误输出（stderr）写入提醒；可设置环境变量 `CTYUN_WARN_CONFIG_CREDENTIALS=0`，或运行 `ctyun config set warn_config_credentials false` 关闭。

安全建议：

- 优先使用环境变量传入 AK/SK；如果写入配置文件，请不要提交到仓库，并限制文件权限。
- 不要把 AK/SK 写入脚本、命令历史或日志。
- 为 `ctyun` 使用最小权限的 IAM 用户 AK/SK，并定期轮换。
- 避免在共享机器或 CI 日志中暴露环境变量。
- 使用 `--debug` 排查请求时，分享日志前仍应再次检查敏感信息。

配置文件适合保存资源池、语言、超时、插件源或测试用端点覆盖，也可以作为 `CTYUN_AK`/`CTYUN_SK` 的后备来源。加载器仍会拒绝不受支持的密钥类字段。

```json
{
  "warn_config_credentials": true,
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "cn-huadong1",
      "language": "zh-CN",
      "ak": "...",
      "sk": "...",
      "timeout_seconds": 20
    }
  }
}
```

可通过非交互命令查看和更新配置：

```sh
ctyun config path
ctyun config show
ctyun config set region cn-huadong1 --profile prod
ctyun config profile use prod
printf '%s\n' "$CTYUN_AK" | ctyun config profile set-secret prod ak --from-stdin
printf '%s\n' "$CTYUN_SK" | ctyun config profile set-secret prod sk --from-stdin
ctyun config reset --yes
```

`ctyun config show` 会把已保存的 AK/SK 显示为 `aa*****dd` 这样的掩码；未配置时保持为空。`ctyun config reset` 会先提示确认；确认后创建备份，再删除当前配置文件。脚本中可使用 `--yes` 或 `-y` 跳过提示。

支持的语言为 `zh-CN`、`en-US` 和 `en-GB`。语言选择顺序为 `--lang`、`CTYUN_LANGUAGE`、配置档案中的 `language`、系统语言；无法匹配时默认 `zh-CN`。

## 插件

新安装的 `ctyun` 只包含核心命令，不会预装产品插件。产品命令来自插件包；完成鉴权、配置和语言设置后，请先安装所需插件：

<details>
<summary>插件列表</summary>

| 名称    | 插件       | 产品       | 版本                                                                                                                                           | 通道       | 质量          |  命令 |  操作 |
|-------|----------|----------|----------------------------------------------------------------------------------------------------------------------------------------------|----------|-------------|----:|----:|
| 弹性云主机 | `ecs`    | `ecs`    | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fecs%2F*&label=release)](../../releases)    | `beta`   | `generated` | 220 | 220 |
| 资源池   | `region` | `region` | [![GitHub Tag](https://img.shields.io/github/v/tag/ArvinZJC/ctyun-cli?filter=releases%2Fplugins%2Fregion%2F*&label=release)](../../releases) | `stable` | `curated`   |   7 |   7 |

质量字段表示插件元数据的整理程度：`generated` 表示工具生成的初稿，`reviewed` 表示已完成基础复核，`curated` 表示作为维护版本持续更新。

</details>

```sh
ctyun plugin search ecs --source auto
ctyun plugin list --available --source auto
ctyun plugin list --available --cols 插件,质量,状态 --filter 状态=可安装 --source auto
ctyun plugin install region ecs --source auto
ctyun plugin install --all --source auto
ctyun plugin list
```

插件管理命令共享这些行为：

- `ctyun plugin search`、`ctyun plugin list --available`、`ctyun plugin install`、`ctyun plugin reinstall` 和 `ctyun plugin update` 都支持 `--source` 和 `--channel`。
- `ctyun plugin list --available` 会显示托管插件及本地安装状态。
- `ctyun plugin list --available` 和 `ctyun plugin search` 可使用 `--channel all` 查看所有插件源通道；安装和更新仍需选择具体通道。
- `ctyun plugin search` 支持模糊搜索，并遵循表格/JSON 输出控制。
- `ctyun plugin reinstall` 会按指定源刷新已安装插件，即使版本号没有变化。
- `--cols`、`--filter` 和 `--sort` 可使用表格中看到的列名，也兼容稳定列键。
- 只有当参数值会被 shell 拆开时才需要加引号，例如使用带空格的英文列名。
- 危险操作默认提示输入 `y/N` 确认；脚本中可使用 `--yes` 或 `-y` 跳过提示。

```sh
ctyun plugin reinstall ecs region --source auto
ctyun plugin reinstall --all --source auto
ctyun plugin update --all --source auto
ctyun plugin update --all --source auto --channel alpha
ctyun plugin remove ecs region --yes
```

安装对应插件后，常用产品命令形态如下：

```sh
ctyun region list
ctyun region list --name 华东1 --cols 资源池ID,资源池名称,地域编号
ctyun ecs instance list --cols "实例 ID,名称,状态"
ctyun ecs instance list --name api-test01
ctyun ecs instance show c5a7966a-88e7-362b-6e11-c2d8fbfc07ca
```

输出控制：

```sh
ctyun ecs instance list --output json
ctyun ecs instance list --table compact
ctyun ecs instance list --table plain
ctyun ecs instance list --no-header
ctyun ecs instance list --filter 状态=running --sort "-实例 ID"
```

## 核心更新

发行包可用后，可通过 `ctyun update` 或 `ctyun upgrade` 检查并更新核心二进制。核心更新只读取 `auto`、`github` 或 `gitee` 托管发布资产；`auto` 先读取 GitHub 发布资产，失败后回退到 Gitee 镜像。签名索引和 SHA-256 校验是信任边界。可通过 `--channel` 选择 `stable`、`beta` 或 `alpha` 通道。

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

## 开发者与贡献者工作流

如果默认 Go 构建缓存不可写（例如在沙盒环境中），先设置仓库内缓存：

```sh
export GOCACHE="$PWD/.cache/go-build"
```

下面示例中的 `<name>`、`<插件命令>` 和其他尖括号值都是占位符，运行前请替换为实际插件名、命令或路径。

开发与调试：

```sh
go run ./cmd/ctyun --offline <插件命令>
go run ./cmd/ctyun --fixture <插件命令>
go run ./cmd/ctyun -O <插件命令>
go run ./cmd/ctyun --debug --offline <插件命令>
```

`--offline`、`--fixture` 和 `-O` 都启用插件内置示例数据，不访问真实天翼云接口，适合本地调试命令形态、表格输出和参数映射。该示例数据模式面向开发和测试场景，因此这些选项都不会出现在常规帮助中。

开发版可用 `--bundled` 从仓库内置插件元数据搜索、列出、安装、重新安装或更新插件。和 `--fixture` 一样，`--bundled` 面向开发和测试场景，不会出现在常规帮助中。

```sh
go run ./cmd/ctyun plugin list --available --bundled
go run ./cmd/ctyun plugin search <name> --bundled
go run ./cmd/ctyun plugin install <name> --bundled
go run ./cmd/ctyun plugin reinstall <name> --bundled
go run ./cmd/ctyun plugin update <name> --bundled
```

测试：

```sh
git ls-files '*.go' | xargs gofmt -w
go test ./...
go test ./internal/cli -run Completion -v
go test ./tools/plugincheck
go run ./tools/coverage
```

插件变更后建议按影响范围验证。先校验被修改的插件，再跑对应离线命令；如果改动影响通用插件加载、命令解析或表格输出，再补充相关 Go 测试。

```sh
go run ./cmd/ctyun plugin lint ./plugins/<name>
go run ./cmd/ctyun --offline <插件命令>

go test ./tools/plugincheck
go test ./internal/cli ./internal/plugin ./internal/output
```

OpenAPI 采集/复核流水线是开发工具，不会暴露为用户命令，也不会进入核心或插件发布包。它从规范化 JSON 输入开始，并把上游证据保存在 `openapi/products/<name>/source.json`：

```sh
go run ./tools/openapi harvest <name> --input path/to/normalized-source.json
go run ./tools/openapi diff <name>
go run ./tools/openapi generate <name>
go run ./tools/openapi review <name>
```

对通过该流水线维护的插件，应跟踪对应的 `source.json` 作为上游证据，并跟踪提升后更新的 `baseline.json` 作为最近一次接受的快照。`draft/`、`changes.md` 和 `review.md` 都是可复现的本地复核输出，默认忽略；需要复核时重新运行 `diff`、`generate` 和 `review` 即可。生成草稿会从 `source.json` 写入 `source_fingerprint`；当草稿通过复核、且 `generated`/`reviewed`/`curated` 质量值准确反映当前整理程度时，运行提升命令会更新插件元数据并推进 `baseline.json`。普通历史由 git 保存。

```sh
go run ./tools/openapi promote <name>
```

发布打包工具会生成核心二进制归档、`core-index.json`、`core-index.sig`、安装脚本、插件归档、`index.json` 和 `index.sig`。开发阶段可通过测试中的假 HTTP 源验证签名和下载逻辑；正式发布资产服务于上面的安装、核心更新和插件更新流程。核心安装和更新入口使用固定发布标签 `core` 作为稳定资产根路径，插件安装和更新入口使用固定发布标签 `plugins` 作为稳定资产根路径；实际版本和通道分别由签名的 `core-index.json` 与 `index.json` 决定。对已有输出目录再次运行打包工具时，它会保留其他通道的现有索引条目，只替换本次重新构建的核心通道或插件名/通道资产，然后重新签名索引；如果为同一核心版本补充平台归档，则会合并平台资产。如需面向用户展示变更记录，仍可另外创建 SemVer 版本标签或发布页。

核心和插件版本必须遵循 Semantic Versioning 2.0.0。发布版本不要加 `v` 前缀。预发布版本使用 `0.1.0-alpha.1` 一类版本号和 `alpha`/`beta` 通道；稳定发布使用 `0.1.0` 一类版本号和 `stable` 通道。`internal/version/version.go` 中的默认值只用于未打包的开发构建，发布打包会覆盖实际版本和通道。

开发和测试专用环境变量：

| 变量                          | 用途                           |
|-----------------------------|------------------------------|
| `CTYUN_INSTALL_BASE_URL`    | 覆盖安装脚本读取的发布根地址，用于本地或临时发布资产验证 |
| `CTYUN_RELEASE_PRIVATE_KEY` | 发布打包工具签名索引使用的私钥              |
| `CTYUN_RELEASE_PUBLIC_KEY`  | 开发构建或私有分发验证中用于核心更新和插件索引验签的公钥 |

```sh
go run ./tools/release --generate-key
export CTYUN_RELEASE_PRIVATE_KEY="<上一步输出的私钥>"
export CTYUN_RELEASE_PUBLIC_KEY="<上一步输出的公钥>"
go run ./tools/release --version 0.2.0 --channel stable --out ./dist/releases --platform "$(go env GOOS)/$(go env GOARCH)"
```

正式发布时，GitHub 仍是源码和 CI 产物的权威来源，Gitee 作为同步镜像提供更稳的国内访问路径。`ctyun` 信任签名公钥和 SHA-256 校验，不信任托管平台本身。

## 友情链接

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli)：另一个非官方天翼云 CLI，使用 Python 编写，适合偏好 Python 生态的用户参考。
