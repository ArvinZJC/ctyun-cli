# 编写与复核插件

[项目首页](../../README.md) · [English](../en/plugin-authoring.md)

## 从一条命令追溯插件定义

以 `ctyun region list --name 保定` 为例。命令从用户输入到接口请求和输出表格，依次涉及下列元数据。新增产品命令时，可以按同样的方式核对各文件的对应关系。

| 资源池插件文件 | 这条命令用到的定义 |
|---|---|
| [plugin.json](../../plugins/region/plugin.json) | 声明插件名、发布通道、质量、兼容的核心版本、API 端点、URI 范围和已采纳来源的指纹 |
| [commands.json](../../plugins/region/commands.json) | 将公开命令路径 `region list` 绑定到操作和表格，将 `--name` 关联到 `regionName` 参数，并指定离线响应文件 |
| [apis.json](../../plugins/region/apis.json) | 声明可重试的 `GET /v4/region/list-regions`，查询字段 `regionName` 从 `$param.regionName` 取值 |
| [tables.json](../../plugins/region/tables.json) | 从 `returnObj.regionList` 取行，将每行的 `regionID` 映射为稳定列键 `region_id`，中文显示为“资源池 ID” |
| [i18n](../../plugins/region/i18n) | 提供各支持语言的命令说明和参数说明 |
| [离线响应文件](../../plugins/region/fixtures/region-list.json) | 保存响应外层结构和样例行，供离线验证 |
| [waiters.json](../../plugins/region/waiters.json) | 保存有明确终止状态的命令轮询定义。资源池插件没有等待器 |

从仓库根目录分别查看表格与 JSON。离线表格会按参数保留匹配行，JSON 会显示记录的完整响应，其中也包含另一行。在线调用时，查询参数交由服务处理。

```sh
go run ./cmd/ctyun plugin lint ./plugins/region
go run ./cmd/ctyun region list --name 保定 --offline
go run ./cmd/ctyun region list --name 保定 --offline --output json
```

使用 `$arg.<name>` 绑定路径参数，核心会将每个值编码为单个 URL 路径段。操作从配置档案读取资源池时，提供可选的 `--region` 供用户覆盖。请求键和载荷值保留官方拼写。按字段在当前操作中的含义选择表格列键和本地化标签。对于需要用户确认的变更操作，在命令元数据中声明确认提示。

## OpenAPI 证据目录维护流程

OpenAPI 证据目录流水线是开发工具，不会暴露为用户命令，也不会进入核心或插件发布包。它从规范化 JSON 输入开始，并把上游证据保存在 `openapi-catalogs/<name>/source.json`：

```sh
go run ./tools/openapi harvest <name> --input path/to/normalized-source.json
go run ./tools/openapi normalize-labels <name>
go run ./tools/openapi diff <name>
go run ./tools/openapi generate <name>
go run ./tools/openapi review <name>
```

[产品覆盖清单](../../openapi-catalogs/product-coverage.json) 将已纳入插件的命令与官方产品目录对照，记录跨产品引用、经过核对的等价接口、范围外接口及已确认的文档障碍。等价覆盖采用目标产品的服务地址和响应契约，不表示旧目录中的响应格式也受到支持。[官方 API 复核记录](../../openapi-catalogs/official-review.json)汇总了最近一次全插件检查、有证据支持的更新、保留的文档缺口和终端节点阻塞项。

产品级记录补充了 [EIP](../../openapi-catalogs/eip/coverage.json)、[PostgreSQL](../../openapi-catalogs/rds-postgresql/coverage.json)、[Redis](../../openapi-catalogs/redis/coverage.json) 和 [MongoDB](../../openapi-catalogs/mongodb/coverage.json) 的接口缺口、源示例修复或等待器限制。

更新插件时，依次完成四步：补充来源证据、生成并复核草稿、检查生命周期等待器、将通过复核的草稿纳入插件。

### 1. 更新来源证据

在 `openapi-catalogs/<name>/` 中维护以下文件：

| 文件                                | 用途                                         | Git 管理方式         |
|-------------------------------------|----------------------------------------------|----------------------|
| `source.json`                       | 当前上游证据、API 范围及用于生成插件的元数据 | 跟踪                 |
| `baseline.json`                     | 最近一次接受的快照，用于校验已纳入插件的内容 | 跟踪；由提升操作更新 |
| `draft/`、`changes.md`、`review.md` | 生成的插件草稿、变更报告及复核结果           | 忽略；需要时重新生成 |

变更尚待复核时，较新的 `source.json` 可以与 `baseline.json` 不同。在提升前，现有插件的来源指纹和 API 范围仍以 `baseline.json` 为准。日常变更历史由 Git 保存。

从天翼云产品和服务目录索引开始。对每项服务，枚举 `ctapiDocumOut/queryByserviceId/{serviceId}` 返回的所有条目，逐一获取菜单，记录上游名称、ID、URL 和操作数量。将已纳入插件的 API ID 与相关菜单的并集比对，包含次级目录和由其他产品负责的操作。获取失败时标记为待核实。只有查全相关菜单后，才能将未找到的操作记为未列出。弃用情况单独记录，以官方通知及其适用范围为准。

文档 URL 中的 `sid` 对应 `product.ctyun_product_id`，存在 `vid` 时将其记入 `product.source_revision`。版本标识可用于追溯来源，即使标识未变，仍需核对所采集的请求与响应契约。

更新 `source.json` 时：

- 用 `product.api_scope` 声明支持的 API URI 范围。生成、复核和提升都必须遵守该范围。
- 用 `request_example` 保存完整请求示例，用 `example` 保存单个参数值。上游未提供可用示例时，在复核中确认缺口，并记录 `example_unavailable`。
- 保留推荐替代 API 的上游证据。只有推荐、没有弃用或下线说明的内容，不能转成弃用警告。

### 2. 生成并复核草稿

使用 `diff`、`generate` 和 `review` 重新生成本地输出。`normalize-labels` 可修复 `source.json` 中的共享技术词大小写和已审核短语；无法可靠修复的标签保持原样，必须处理后才能通过复核。

复核对象是可运行的插件草稿，而不只是来源目录。检查 API 契约、命令绑定、帮助、示例和响应夹具：

- 普通 JSON 夹具和 HTTP 响应夹具必须与已采集的来源证据一致。数据缺失或被改动会阻止复核通过。
- 表格的行选择器、列键、列取值路径和列弃用信息须与来源目录一致。复核时可以调整布局和默认显示的列。
- 示例必须比 Usage 行提供更多信息，例如具体参数、有意义的选项、结构化值或其他行为。只重复命令路径的示例，包括未解析的路径占位符形式，都会被拒绝。
- 检查草稿中的每条命令是否提供了所有支持语言的描述，改写机械拼接的英文描述。检查示例是否缺少必填选项、使用了未声明的选项，或提供了与参数类型不符的值。
- 只有推荐目标能解析到已跟踪且已纳入插件的 CLI 命令时，才生成推荐帮助；否则在 `source.json` 中保留未解析的证据。插件加载时，跨插件引用是软依赖；发布检查则要求目标精确匹配、未被弃用，并且不存在推荐循环。

如果 API 的响应契约明确，但没有可用的成功响应示例，用 `fixture_unavailable` 记录原因。保留命令、响应校验和原始证据，不编造成功夹具。该命令不能使用 `--offline` 或 `--fixture`，缺失的示例也不能作为等待器证据。下载接口应在响应分支声明文档规定的成功 `media_type`，避免把 HTTP 200 错误响应保存为文件。

### 3. 检查生命周期等待器

每次复核时，检查异步操作是否有安全的查询命令和明确记载的终止状态。等待器只能绑定到非危险且明确允许重试的查询操作。状态证据或资源标识参数不足时记录缺口，不要猜测操作何时完成。

等待器定义及上游证据保存在目录的 `waiters` 映射中。草稿中的等待器定义必须与来源目录一致，提升会保留已接受的定义。修改定义后，应重新生成并复核草稿，再执行提升。使用真实插件包进行离线测试，检查响应结构、终止结果以及命令级帮助和补全。

<details>
<summary>等待器元数据参考</summary>

| 字段                               | 要求                                                                                     |
|------------------------------------|------------------------------------------------------------------------------------------|
| `commands`                         | 显式绑定到精确命令 ID                                                                    |
| `path` 或 `xml_path`               | JSON 状态路径，或精确选择一个标量元素的绝对 XML 命名空间路径；XML 路径不能使用集合选择器 |
| `success`、`failure`               | 文档规定的终止值；未记录失败条件时，两个失败状态字段均留空                                           |
| `success_values`、`failure_values` | 需要时补充其他有文档依据的终止值                                                         |
| `max_attempts`、`interval_seconds` | 正数尝试次数上限和轮询间隔（秒）                                                         |
| `evidence`                         | 状态定义的来源和含义                                                                     |

对于集合响应，添加 `selector`：`path` 定位集合，`key` 标识各行，`value` 引用现有的 `$arg.<name>` 或标量字符串/整数 `$param.<name>`。状态路径相对于选中的行。检查每条已采集记录，精确保留数值标识，并让空值或未知状态继续等待。没有可用的唯一资源标识参数时，记录阻塞原因。

</details>

### 4. 将通过复核的草稿纳入插件

提升前，确认草稿已通过复核，且声明的质量（`generated`、`reviewed` 或 `curated`）准确反映插件的整理程度。

生成过程根据 `source.json` 计算 `source_fingerprint`。对于已有插件，保留已提升清单中的版本、通道、质量和核心兼容范围，并在需要时提高核心最低版本：

| 元数据功能 | 核心最低版本 |
|------------|--------------|
| 基础生成插件包 | `0.3.1` |
| 非字符串类型的请求体参数 | `0.4.0` |
| 等待器、显式传输策略、原生存储、文件下载或 XML 响应表格 | `0.5.0` |
| 独立的请求头、查询参数和请求体绑定 | `0.5.1` |

重新生成不能降低已有的兼容要求，也不能重置插件版本、通道或质量。提升操作会更新插件元数据，并将 `baseline.json` 推进到已接受的来源快照：

```sh
go run ./tools/openapi promote <name>
```
