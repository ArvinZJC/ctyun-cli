# 查找与管理插件

[项目首页](../../README.md) · [English](../en/plugins.md)

新安装的 `ctyun` 包含核心命令。你可以在配置云凭据之前浏览并安装所需的产品插件。

## 选择插件与通道

在[产品插件目录](catalogue.md)中查找产品对应的命令前缀和发布通道。例如，资源池插件使用 `stable`，ECS 插件使用 `beta`。运行 `ctyun plugin list --available --channel all` 可查看所选镜像已发布的插件，运行 `ctyun plugin list` 可查看当前用户已经安装的插件。默认安装目录为 `~/.ctyun/plugins`。

## 安装、修复或更新插件

```sh
ctyun plugin search ecs --source auto --channel beta
ctyun plugin list --available --source auto
ctyun --lang zh-CN plugin list --available --cols 插件,质量,状态 --filter 状态=可安装 --source auto
ctyun plugin install region --source auto
ctyun plugin install ecs --source auto --channel beta
ctyun plugin list
```

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
```

要安装所选通道中所有尚未安装的插件，使用 `ctyun plugin install --all` 并指定通道。要移除指定插件，运行 `ctyun plugin remove ecs region` 并按提示确认。在脚本中已明确需要移除时，可加 `--yes`。

## 检查更新或排查找不到命令的问题

- 运行 `ctyun plugin list --updates --channel beta` 检查 beta 插件更新。要仅更新 ECS 插件，运行 `ctyun plugin update ecs --channel beta`。
- 搜索不到产品时，使用 `--channel all` 再查一次。如果帮助中找不到产品命令，先核对本地插件列表，再按目录所列通道安装相应插件。
- 安装时提示核心版本不兼容，可运行 `ctyun --version` 和 `ctyun update --check` 检查版本与更新。无法获取索引时，运行 `ctyun doctor network --source github` 检查该镜像的连通性，或指定 `gitee` 检查另一镜像。
