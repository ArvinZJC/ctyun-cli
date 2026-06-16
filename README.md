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
- 支持 i18n：核心帮助、错误提示、插件名称、命令说明和表格列都可以本地化。

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

实时请求只从进程环境读取 AK/SK：

```sh
export CTYUN_AK=...
export CTYUN_SK=...
```

安全建议：

- 不要把 AK/SK 写入仓库、脚本、配置文件、命令历史或日志。
- 为 `ctyun` 使用最小权限的 IAM 用户 AK/SK，并定期轮换。
- 避免在共享机器或 CI 日志中暴露环境变量。
- 使用 `--debug` 排查请求时，分享日志前仍应再次检查敏感信息。

配置文件适合保存非密钥默认值，例如资源池、语言、超时、插件源或测试用 endpoint 覆盖。加载器会拒绝看起来像 AK/SK 或 secret 的字段。

```json
{
  "active_profile": "prod",
  "profiles": {
    "prod": {
      "region": "cn-huadong1",
      "language": "zh-CN",
      "timeout_seconds": 20
    }
  }
}
```

配置查找顺序为 `--config`、`CTYUN_CONFIG`、`~/.ctyun/config.json`。使用 `--profile` 可选择命名 profile。

支持的语言为 `zh-CN`、`en-US` 和 `en-GB`。语言选择顺序为 `--lang`、`CTYUN_LANGUAGE`、profile 中的 `language`、系统语言；无法匹配时默认 `zh-CN`。

## 插件

产品命令来自插件包。当前以插件形式支持 ECS 和 Region 查询；对应插件位于
`plugins/ecs` 和 `plugins/region`，仍在开发完善中。

```sh
ctyun plugin lint ./plugins/ecs
ctyun plugin install ./plugins/ecs
ctyun plugin install ./ctyun-plugin-ecs-0.4.2.tar.gz
ctyun plugin list
ctyun plugin remove ecs
```

插件源可以是本地目录或签名的 HTTP(S) 索引。解析顺序为 `--registry`、`CTYUN_REGISTRY_URL`、当前 profile 的 registry 配置。

```sh
ctyun plugin search --registry ./registry
ctyun plugin install ecs --registry ./registry
ctyun plugin update --all --registry ./registry
```

## 开发者与贡献者工作流

如果默认 Go build cache 不可写（例如在沙盒环境中），先设置仓库内缓存：

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

`--offline`、`--fixture` 和 `-O` 都启用插件内置 fixture，不访问真实天翼云接口，适合本地调试命令形态、表格输出和参数映射。该 fixture 模式面向开发和测试场景，因此这些选项都不会出现在常规帮助中。

测试：

```sh
git ls-files '*.go' | xargs gofmt -w
GOCACHE="$PWD/.cache/go-build" go test ./...
GOCACHE="$PWD/.cache/go-build" go test ./internal/cli -run Completion -v
GOCACHE="$PWD/.cache/go-build" go run ./tools/coverage
```

插件变更后建议按影响范围验证。先 lint 被修改的插件，再跑对应离线命令；如果
改动影响通用插件加载、命令解析或表格输出，再补充相关 Go 测试。

```sh
go run ./cmd/ctyun plugin lint ./plugins/ecs
go run ./cmd/ctyun --offline ecs instance list

go run ./cmd/ctyun plugin lint ./plugins/region
go run ./cmd/ctyun --offline region list

GOCACHE="$PWD/.cache/go-build" go test ./internal/cli ./internal/plugin ./internal/output
```

## 友情链接

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli)：另一个非官方天翼云 CLI，使用 Python 编写，适合偏好 Python 生态的用户参考。
