# ctyun-cli

[![Release version](https://img.shields.io/badge/dynamic/json?url=https%3A%2F%2Fgithub.com%2FArvinZJC%2Fctyun-cli%2Freleases%2Fdownload%2Fcore%2Fcore-index.json&query=%24.versions.stable&label=release&color=blue)](https://github.com/ArvinZJC/ctyun-cli/releases/tag/core)
![GitHub commit activity](https://img.shields.io/github/commit-activity/m/ArvinZJC/ctyun-cli)
[![GitHub License](https://img.shields.io/github/license/ArvinZJC/ctyun-cli?label=licence)](./LICENCE)

简体中文 | [English](./README-EN.md)

`ctyun` 是用于查询和管理天翼云资源的非官方 Go 命令行工具。按需安装产品插件后，可以在终端中查看表格，也可以输出 JSON 供脚本处理。仓库和软件包名为 `ctyun-cli`。天翼云官方 CLI 的命令名为 `ctyun-cli`，两者可以共存。

本项目覆盖自研池（一类节点）的客户侧 API。核心帮助、错误、运行时警告、插件名称、命令说明和表格标签均支持中英文。

## 安装

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

如果安装脚本输出了设置 `PATH` 的命令，请在准备使用 `ctyun` 的终端中执行。Gitee 下载方式、自定义安装目录和常见问题见[安装指南](docs/zh-CN/installation.md)。

## 第一次查询

先检查版本，安装资源池插件，再查看命令帮助。安装插件需要联网，查看帮助无需配置云账号。

```sh
ctyun --version
ctyun plugin install region
ctyun region list --help
```

将有权查询资源池的账号凭据设为当前终端的 `CTYUN_AK` 和 `CTYUN_SK`，然后运行：

```sh
ctyun region list
```

表格列出当前账号可见的资源池。脚本可以使用 `ctyun region list --output json` 读取 API 响应。凭据设置方式见[凭据与配置档案](docs/zh-CN/configuration.md)。

## 文档

| 我想做什么 | 指南 |
|---|---|
| 安装、升级或卸载 CLI | [安装指南](docs/zh-CN/installation.md) |
| 设置凭据、选择配置档案或排查连接问题 | [配置指南](docs/zh-CN/configuration.md) |
| 查找和更新插件，确认产品覆盖范围 | [插件管理](docs/zh-CN/plugins.md) · [产品插件目录](docs/zh-CN/catalogue.md) |
| 查看命令帮助、筛选表格、编写查询脚本或等待资源就绪 | [运行产品命令](docs/zh-CN/commands.md) |
| 使用存储凭证上传或下载文件 | [存储凭证与文件传输](docs/zh-CN/storage.md) |

### 参与贡献

- [核心开发](docs/zh-CN/development.md)：了解请求处理流程、代码归属、离线测试和提交前检查。
- [插件编写与复核](docs/zh-CN/plugin-authoring.md)：以一条实际命令为例，从上游接口资料追溯到可运行的插件。
- [发布维护](docs/zh-CN/releases.md)：管理版本、验证安装脚本、签名并发布到 GitHub 和 Gitee。

如果正在阅读二进制发布包中的 README，可前往[源码仓库](https://github.com/ArvinZJC/ctyun-cli)查看完整指南。

## 友情链接

- [fengyucn/ctyun-cli](https://github.com/fengyucn/ctyun-cli)：另一个非官方天翼云 CLI，使用 Python 编写，适合偏好 Python 生态的用户参考。
