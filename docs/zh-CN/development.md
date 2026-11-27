# 开发与测试 ctyun

[项目首页](../../README.md) · [English](../en/development.md)

## 从一条命令了解核心执行流程

从仓库根目录运行以下命令。它需要 Go 和项目依赖，使用资源池插件随附的响应样本。输出应包含保定这一行，地域编号为 `cn-baoding-1`。

```sh
go run ./cmd/ctyun region list --name 保定 --cols region_name,region_code --offline
```

| 要修改的行为 | 从哪里开始 | 重点验证 |
|---|---|---|
| 命令分派、选项或帮助 | [CLI 入口](../../internal/cli/cli.go)、[帮助](../../internal/cli/help.go) | 命令组帮助、末级命令的参数错误，以及空输入和已输入部分选项名时的补全 |
| 配置或语言选择 | [配置解析](../../internal/config/resolution.go) | 生效值、取值来源以及诊断信息中的凭据保护 |
| 请求构造或鉴权 | [命令绑定](../../internal/cli/command_bindings.go)、[客户端](../../internal/client)、[签名](../../internal/signing) | 按接口契约核对查询参数、请求体、请求头和路径编码 |
| 输出 | [命令表格](../../internal/cli/command_tables.go)、[渲染器](../../internal/output) | 本地化列名、稳定列键，以及交互终端与重定向时的显示宽度 |
| 插件发现或安装 | [插件索引](../../internal/registry)、[分发校验](../../internal/distribution)、[本地安装](../../internal/plugin/install.go) | 索引签名、校验和、清单身份、兼容性与安全解包 |
| 产品命令或等待器 | [插件编写与复核](plugin-authoring.md) | 来源证据、正式插件元数据和真实插件的离线测试 |

修改核心行为时，使用独立构造的元数据，通过 `cli.Execute` 或 `cli.Run` 测试公开命令。修改产品行为时，在 [tools/plugincheck](../../tools/plugincheck) 中使用仓库内对应的插件包编写回归用例。离线测试能验证命令和已记录的响应结构。要确认端点连通性、账号权限和服务当前返回的数据，还需执行在线查询。

## 准备提交

解析、帮助和补全共用同一份选项声明。帮助文案写入帮助词条，运行时诊断写入消息词条，产品名称和表格标签写入插件语言文件或表格定义。请求字段名等机器值在各语言中保持一致。公开行为有变更时，同步更新两种语言的指南。代码注释、文件行数和提交标题要求见[项目规则](../../AGENTS.md)，选择变更日志的依据见[发布指南](releases.md)。

## Go 工具链

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

## 构建与测试

要构建并运行独立的开发版可执行文件，可在仓库根目录执行以下命令。产物放在临时目录，会在子 Shell 退出时清理。

```sh
(
  build_dir=$(mktemp -d) || exit 1
  trap 'rm -rf "$build_dir"' EXIT
  go build -o "$build_dir/ctyun" ./cmd/ctyun && "$build_dir/ctyun" --version
)
```

```sh
go run ./cmd/ctyun region list --offline
go run ./cmd/ctyun region list --fixture
```

`--offline` 和 `--fixture` 都启用插件内置示例数据，不访问真实天翼云接口，可用于核对命令用法、样例响应和表格输出。它们是仅供开发版使用的产品命令长选项，必须放在完整产品命令路径之后，不是全局选项，也没有短选项。发布版不会识别或暴露这些开发选项。

需要查看实际请求时，使用有权查询资源池的凭据运行 `go run ./cmd/ctyun --debug region list`。HTTP 调试信息写入标准错误。离线模式读取本地响应样本，不会产生 HTTP 调试记录。

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
