# 运行产品命令

[项目首页](../../README.md) · [English](../en/commands.md)

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

## 查看当前安装版本的帮助

运行 `ctyun region` 查看子命令，再运行 `ctyun region list --help` 查看这条查询支持的选项和列名。用法中的 `{name}` 表示位置参数，`<value>` 表示选项值，方括号内的输入可以省略。命令需要资源池时，从 `ctyun region list` 的结果中选择 ID，写入配置档案，或在提供 `--region` 的命令上直接指定。

要在当前 Bash 会话启用补全，运行 `source <(ctyun completion bash)`。Fish 用户可运行 `ctyun completion fish | source`。Zsh 用户可将 `ctyun completion zsh` 的输出保存为 `fpath` 目录下的 `_ctyun` 文件，再调用 `compinit`。PowerShell 用户可运行 `ctyun completion powershell | Out-String | Invoke-Expression` 加载生成的脚本。产品命令的补全候选项取决于当前可用的插件。

## 产品命令

危险操作默认提示输入 `y/N` 确认；脚本中可使用 `--yes` 或 `-y` 跳过提示。

运行下列示例前，先安装资源池插件和 ECS 插件。查询 ECS 时，需要在配置档案中设置资源池，或在每条命令上加 `--region <id>`。请将示例实例 ID 换成当前账号查询到的 ID。

```sh
ctyun region list
ctyun --lang zh-CN region list --name 华东1 --cols "资源池 ID,资源池名称,地域编号"
ctyun --lang zh-CN ecs instance list --cols "实例 ID,名称,状态"
ctyun ecs instance list --name api-test01
ctyun ecs instance show c5a7966a-88e7-362b-6e11-c2d8fbfc07ca
```

## 输出与筛选

- `--cols`、`--filter` 和 `--sort` 可使用表格中看到的列名，也兼容稳定列键。列名有歧义时，可使用帮助中显示的完整列名或稳定列键。
- 列名含空格或筛选表达式含 shell 特殊字符时，请加引号。例如，`--filter '名称=*'` 匹配名称显示值为字面字符 `*` 的行。

```sh
ctyun ecs instance list --output json
ctyun ecs instance list --table compact
ctyun ecs instance list --table plain
ctyun ecs instance list --no-header
ctyun --lang zh-CN ecs instance list --filter 状态=running --sort "-实例 ID"
```

交互式表格会按终端显示宽度计算中文、英文、Emoji 等 Unicode 内容，并优先在空白或常见机器值分隔符处换行；输出重定向或通过管道传递时则保留自然宽度。`bordered`、`compact` 和 `plain` 样式共用同一套列宽计算和换行规则。

## 等待资源或任务状态

等待器需要核心版本 `>=0.5.0 <1.0.0`。在查询命令中使用 `--wait <waiter>`，轮询直到观察到文档定义的目标状态。命令帮助会列出适用的等待器，Shell 补全提供相同选项。发送请求前会检查等待器，拒绝不适用的命令和未声明安全轮询元数据的操作。轮询会重复执行查询命令，因此应使用之前变更操作返回的资源或任务标识。

```sh
ctyun ecs instance show {instance_id} --wait ecs.instance.running
ctyun vbs backup show --backup-id <backup_id> --wait vbs.backup.available
ctyun hpfs dataflow-task show --task-id <task_id> --wait hpfs.dataflow-task.completed
ctyun evs snapshot list --snapshot-id <snapshot_id> --wait evs.snapshot.available
ctyun ims image show {image_id} --wait ims.image.active
```

等待集合中的单个资源时，必须提供命令帮助要求的资源标识。每次响应都按该标识精确匹配：找不到资源时继续等待，匹配到多个资源时报告错误。

首次响应计为一次轮询。每个等待器规定轮询次数上限和两次轮询之间的等待时间。一次轮询中的查询可能因重试发出多个 HTTP 请求，每个请求单独计算超时。在 64 位平台上，轮询间隔和单次请求超时的上限均为 9,223,372,036 秒。状态值为空或未知时会继续等待，直到达到轮询上限。等待器会输出失败或超时状态，命令退出码保持不变。JSON 模式下，首次响应写入标准输出，等待器状态写入标准错误。

## 在脚本中使用查询结果

例如，`ctyun region list` 的 `--name` 选项会将查询参数发送给服务端。表格选项则处理本次请求返回的行。在中文界面中，`--filter '资源池名称=保定'` 按显示值精确匹配，`--filter '资源池名称!=保定'` 排除匹配行。排序按显示字符串比较，在列名前加 `-` 可倒序排列。需要跨界面语言运行的脚本可使用该列的稳定键 `region_name`。

JSON 保留 API 的响应字段。安装资源池插件、配置好凭据并安装 `jq` 后，以下 Bash 管道逐行输出资源池 ID 和名称：

```bash
set -o pipefail
ctyun region list --output json | jq -r '.returnObj.regionList[] | [.regionID, .regionName] | @tsv'
```

脚本在等待资源就绪后还有后续操作时，请先判断等待器输出的最终状态。仅凭进程成功退出无法判断资源是否已到达目标状态。
