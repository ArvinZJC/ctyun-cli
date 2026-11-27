# 凭据、配置档案与诊断

[项目首页](../../README.md) · [English](../en/configuration.md)

实时请求优先从进程环境读取 AK/SK：

```sh
export CTYUN_AK=...
export CTYUN_SK=...
```

如果某个凭据环境变量缺失，`ctyun` 会先从所选配置档案读取对应的 `ak` 或 `sk`，再回退到全局配置。实时请求只要使用了配置中的任一凭据，`ctyun` 就会向标准错误输出（stderr）写入提醒。设置 `CTYUN_WARN_CONFIG_CREDENTIALS=0` 可关闭当前进程的提醒。要默认关闭提醒，请运行 `ctyun config set warn_config_credentials false`。如果某个配置档案显式开启了提醒，可在该命令末尾加上 `--profile <name>` 来修改该档案的设置。

## 配置与配置档案

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
```

如需将凭据保存到该配置档案，使用标准输入传入密钥。以下命令会将当前终端环境中已有的凭据写入配置文件。

```sh
printf '%s\n' "$CTYUN_AK" | ctyun config profile set-secret prod ak --from-stdin
printf '%s\n' "$CTYUN_SK" | ctyun config profile set-secret prod sk --from-stdin
```

`ctyun config show` 显示已存储的 JSON，并把已保存的 AK/SK 显示为 `aa*****dd` 这样的掩码；未配置的值会被省略。`ctyun config explain` 则显示生效的基础设置，以及每个值最终采用的来源。敏感设置只说明是否已配置，不会显示、掩码、指纹化或以其他方式派生 AK/SK。

使用 `ctyun doctor local` 检查配置、凭据、资源池设置和已安装插件，不访问网络或修改本地状态。它会输出所有独立检查结果；存在失败项时退出码为 1，只有警告或跳过项时为 0。使用 `ctyun doctor network` 检查在线下载源和 API 端点。

确认后，`ctyun config reset` 会在配置文件所在目录创建备份，再删除原文件。备份依次尝试 `.bak`、`.bak.2`、`.bak.3` 等后缀，使用第一个未被占用的名称。脚本中可使用 `--yes` 或 `-y` 跳过提示。

如果另一个 `ctyun` 进程在保存或重置期间修改了配置，命令可能会提示冲突。请先运行 `ctyun config show` 查看当前配置，再决定是否重试。

## 语言

支持的语言为 `zh-CN`、`en-US` 和 `en-GB`。语言选择顺序为 `--lang`、`CTYUN_LANGUAGE`、配置档案中的 `language`、系统语言；无法匹配时默认 `zh-CN`。

## 切换不同环境的配置档案

经常使用多个资源池或环境时，可以分别建立配置档案。请将示例 ID 换成当前账号可见的资源池 ID。以下命令创建或修改名为 `staging` 的档案，仅在一次检查中指定该档案，最后再将其设为后续命令的默认档案。

```sh
ctyun config profile set staging region 81f7728662dd11ec810800155d307d5b
ctyun config profile list
ctyun --profile staging config explain region
ctyun config profile use staging
```

切换档案后若仍像在使用另一个账号，运行 `ctyun --profile staging config explain ak`，再将键换为 `sk` 查询来源。环境变量的优先级仍高于档案配置。Windows PowerShell 使用 `$env:CTYUN_AK` 和 `$env:CTYUN_SK` 访问环境变量，凭据可通过当前终端或 CI 作业的密钥管理方式注入。

## 根据现象排查

| 现象 | 检查方法 |
|---|---|
| 生效的资源池、配置档案或文件与预期不符 | 在运行命令的同一个终端执行 `ctyun config path` 和 `ctyun config explain` |
| 本地设置或已安装插件异常 | 运行 `ctyun doctor local`，逐项查看失败的检查 |
| 插件源或 API 端点无法访问 | 运行 `ctyun doctor network`，使用 `--source github` 或 `--source gitee` 可单独检查镜像 |
| 服务可达，但拒绝查询 | 核对凭据来源、账号对该操作的权限，以及服务在该资源池的可用性 |
