# ctyun-cli

![GitHub commit activity](https://img.shields.io/github/commit-activity/m/ArvinZJC/ctyun-cli)
[![GitHub](https://img.shields.io/github/license/ArvinZJC/ctyun-cli)](./LICENCE)

简体中文 | [English](./README-EN.md)

> [!WARNING]
> 假装夜以继日开发中，再迭代 yi 次就发布了。

`ctyun-cli` 是仓库名，`ctyun` 是命令行工具名。这是一个使用 Go 编写、基于天翼云 OpenAPI 的非官方 CLI：插件化，体验优先，面向终端里的天翼云资源查询和管理。（目前天翼云没有官方 CLI。）

天翼云官方 Go SDK 名为 `ctyun-go-sdk`，但产品覆盖有限，且未公开发布；如需官方 SDK，可向天翼云提交工单获取。本项目不是 SDK，而是面向用户操作流程的命令行工具。

## 使用前须知

- 请先开通要操作的天翼云服务，并了解对应 OpenAPI 的基本用法。
- 本项目基于天翼云 OpenAPI 的 C 端接口，即消费者/客户侧接口。
- 不支持 B 端接口，即业务/运营侧接口。
- 支持一类节点，即自研池；不支持二类节点，即合营池。
- 仅支持 AK/SK 鉴权，因为天翼云 OpenAPI 当前只支持 AK/SK。
- 暂未提供发行包；请先从源码运行。

OpenAPI 文档入口：[天翼云 OpenAPI 文档](https://eop.ctyun.cn/ebp/ctapiDocument/index)。其中的 API 文档就是这里提到的 C 端接口文档。

## 亮点

- 默认表格输出，适合人工查看，支持中英文内容宽度对齐。
- 支持 `--output json`，便于脚本和其他工具处理。
- 产品命令由插件元数据提供，核心 CLI 不需要为每个产品写专门分支。
- 插件可声明请求方法、路径、参数、表格列、示例、等待器和危险操作确认。
- 支持国际化：核心帮助、错误提示、运行时提醒、插件名称、命令说明和表格列都可以本地化。

## 快速开始

从源码运行：

```sh
go run ./cmd/ctyun version
go run ./cmd/ctyun help
go run ./cmd/ctyun --offline region list
go run ./cmd/ctyun --offline ecs instance list
```

如果已经安装为 `ctyun`，常用命令形态如下：

```sh
ctyun region list
ctyun region list --name 华东1 --cols region_id,region_name,region_code
ctyun ecs instance list --cols instance_id,name,status
ctyun ecs instance show ins-demo-1
ctyun --yes ecs instance start ins-demo-1
ctyun --wait ecs.instance.running ecs instance show ins-demo-1
```

输出控制：

```sh
ctyun ecs instance list --output json
ctyun ecs instance list --table compact
ctyun ecs instance list --table plain
ctyun ecs instance list --no-header
ctyun ecs instance list --filter status=running --sort -instance_id
```

## 鉴权、配置与语言

配置文件查找顺序为 `--config`、`CTYUN_CONFIG`、`~/.ctyun/config.json`；`--profile` 会覆盖 `active_profile`。除这类用于定位配置文件的选项外，运行时设置遵循“命令行选项、环境变量、当前配置档案、支持的全局配置后备”的顺序；同一设置同时出现在环境变量和配置中时，环境变量优先。`CTYUN_CONFIG` 是例外：它用于找到配置文件，因此不会再从配置文件读取自身的后备值。

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

`ctyun config show` 会把已保存的 AK/SK 显示为 `aa*****dd` 这样的掩码；未配置时保持为空。`ctyun config reset --yes` 会先创建备份，再删除当前配置文件。

支持的语言为 `zh-CN`、`en-US` 和 `en-GB`。语言选择顺序为 `--lang`、`CTYUN_LANGUAGE`、配置档案中的 `language`、系统语言；无法匹配时默认 `zh-CN`。

## 插件

产品命令来自插件包。当前以插件形式支持 ECS 和地域查询；对应插件位于
`plugins/ecs` 和 `plugins/region`，仍在开发完善中。

```sh
ctyun plugin lint ./plugins/ecs
ctyun plugin search ecs --source auto
ctyun plugin install ecs --source auto
ctyun plugin list
ctyun plugin remove ecs
```

插件更新使用与核心更新一致的托管源：`auto`、`github` 或 `gitee`。`auto` 先读取 GitHub 发布资产，失败后回退到 Gitee 镜像；签名索引和 SHA-256 校验仍是信任边界。

```sh
ctyun plugin update --all --source auto
```

## 开发者与贡献者工作流

如果默认 Go 构建缓存不可写（例如在沙盒环境中），先设置仓库内缓存：

```sh
export GOCACHE="$PWD/.cache/go-build"
```

开发与调试：

```sh
go run ./cmd/ctyun version
go run ./cmd/ctyun help ecs instance list
go run ./cmd/ctyun --offline region list
go run ./cmd/ctyun --fixture region list
go run ./cmd/ctyun -O region list
go run ./cmd/ctyun --offline ecs instance list
go run ./cmd/ctyun --debug --offline ecs instance list
go run ./cmd/ctyun completion zsh
go run ./cmd/ctyun doctor network
```

`--offline`、`--fixture` 和 `-O` 都启用插件内置示例数据，不访问真实天翼云接口，适合本地调试命令形态、表格输出和参数映射。该示例数据模式面向开发和测试场景，因此这些选项都不会出现在常规帮助中。

开发版可用 `--bundled` 从仓库内置插件元数据安装或更新插件。和 `--fixture`
一样，`--bundled` 面向开发和测试场景，不会出现在常规帮助中。

```sh
go run ./cmd/ctyun plugin install ecs --bundled
go run ./cmd/ctyun plugin update ecs --bundled
```

测试：

```sh
git ls-files '*.go' | xargs gofmt -w
GOCACHE="$PWD/.cache/go-build" go test ./...
GOCACHE="$PWD/.cache/go-build" go test ./internal/cli -run Completion -v
GOCACHE="$PWD/.cache/go-build" go run ./tools/coverage
```

插件变更后建议按影响范围验证。先校验被修改的插件，再跑对应离线命令；如果
改动影响通用插件加载、命令解析或表格输出，再补充相关 Go 测试。

```sh
go run ./cmd/ctyun plugin lint ./plugins/ecs
go run ./cmd/ctyun --offline ecs instance list

go run ./cmd/ctyun plugin lint ./plugins/region
go run ./cmd/ctyun --offline region list

GOCACHE="$PWD/.cache/go-build" go test ./internal/cli ./internal/plugin ./internal/output
```

发布打包工具会生成核心二进制归档、`core-index.json` 和 `core-index.sig`。真实自升级只从 `auto`、`github` 或 `gitee` 读取托管发布资产；开发阶段可通过测试中的假 HTTP 源验证签名和下载逻辑。

```sh
go run ./tools/release --generate-key
export CTYUN_RELEASE_PRIVATE_KEY="<上一步输出的私钥>"
export CTYUN_RELEASE_PUBLIC_KEY="<上一步输出的公钥>"
go run ./tools/release --version 0.2.0 --channel stable --out ./dist/releases --platform "$(go env GOOS)/$(go env GOARCH)"
```

正式发布时，GitHub 仍是源码和 CI 产物的权威来源，Gitee 作为同步镜像提供更稳的国内访问路径。`ctyun` 信任签名公钥和 SHA-256 校验，不信任托管平台本身。

## 友情链接

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli)：另一个非官方天翼云 CLI，使用 Python 编写，适合偏好 Python 生态的用户参考。
