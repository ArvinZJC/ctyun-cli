# 准备与验证发布

[项目首页](../../README.md) · [English](../en/releases.md)

## 确定需要更新的版本

将核心以及各插件的发布内容分别与各自上一个版本标签比较。对插件，比较范围包括新增文件和清单中的来源指纹。确认拟用的版本标签尚未被占用。仅修改插件元数据、仓库工具或文档时，这些变更可以独立于核心版本发布。

已交付的可执行文件发生变化时，记录到[核心变更日志](../../CHANGELOG.md)。各插件独立维护英文变更日志，可参考[资源池插件变更日志](../../plugins/region/CHANGELOG.md)。开发期间将变化汇总到 `Unreleased`，准备发布时用版本号与日期替换该标题。按影响范围更新清单、兼容要求、示例以及两种语言的[产品插件目录](catalogue.md)。

## 验证安装脚本

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

核对可执行文件输出的版本与安装成功提示是否一致。要单独验证某个发布源，可在容器镜像名称前加入 `-e CTYUN_INSTALL_SOURCE=github` 或 `-e CTYUN_INSTALL_SOURCE=gitee`。要在 macOS 或 Windows 上验证已发布的安装脚本，可将 `CTYUN_INSTALL_DIR` 指向新建的临时目录，运行[安装指南](installation.md)中对应平台的命令，再用该目录中的可执行文件运行 `--version` 和 `--help`。

## 打包与发布

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

输出目录根部的插件索引和归档用于 GitHub；`gitee/index.json` 与 `gitee/index.sig` 用于覆盖 Gitee 固定 `plugins` 发布页上的同名文件。`gitee/releases.json` 是发布操作清单，记录每个归档的插件名、版本、通道、目标标签、校验和及预期下载 URL，不作为用户下载资产上传。先确认清单中已有的 Gitee 版本发布和归档与记录一致，并发布缺少的不可变版本和归档，再更新固定 `plugins` 发布页的签名索引。

核心和插件版本必须遵循 Semantic Versioning 2.0.0。发布版本不要加 `v` 前缀。预发布版本使用 `0.1.0-alpha.1` 一类版本号和 `alpha`/`beta` 通道；稳定发布使用 `0.1.0` 一类版本号和 `stable` 通道。`internal/version/version.go` 中的默认值只用于未打包的开发构建，发布打包会覆盖实际版本和通道。

签名索引还包含由最终合并条目生成的 `versions` 映射：核心使用 `versions.<channel>`，插件使用 `versions.<name>.<channel>`，每个值均为对应通道的最高语义版本。README 和产品插件目录中的动态徽章读取这些映射；CLI 仍通过原有数组选择下载产物。

打包与分发验证使用的环境变量：

| 变量                        | 用途                                                     |
|-----------------------------|----------------------------------------------------------|
| `CTYUN_INSTALL_BASE_URL`    | 覆盖安装脚本读取的发布根地址，用于本地或临时发布资产验证 |
| `CTYUN_RELEASE_PRIVATE_KEY` | 发布打包工具签名索引使用的私钥                           |
| `CTYUN_RELEASE_PUBLIC_KEY`  | 设置后将覆盖内置公钥，用于验证核心和插件索引；发布版也会读取该变量 |

本地演练时，可生成临时签名密钥，将当前平台的发布包写入新建的临时目录。打包工具会从私钥派生公钥，并将公钥嵌入二进制文件。为已有发布渠道打包时，请使用已安装客户端所信任公钥对应的发布私钥。

```sh
go run ./tools/release --generate-key
release_dir=$(mktemp -d) &&
  CTYUN_RELEASE_PRIVATE_KEY="<上一步输出的私钥>" \
  go run ./tools/release --version 0.5.2 --channel stable --out "$release_dir" --gitee-plugin-download-root "https://gitee.com/ArvinZJC/ctyun-cli/releases/download" --platform "$(go env GOOS)/$(go env GOARCH)"
```

正式发布时，GitHub 仍是源码和 CI 产物的权威来源，Gitee 作为同步镜像提供更稳的国内访问路径。已安装的 `ctyun` 使用签名公钥验证托管的核心及插件索引，并检查归档 SHA-256；引导安装脚本采用[安装指南](installation.md)中说明的信任方式。
