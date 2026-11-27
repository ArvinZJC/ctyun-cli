# 存储凭证与文件传输

[项目首页](../../README.md) · [English](../en/storage.md)

## 原生存储凭证

`media-storage` 和 `classic-object-storage` 插件要求核心版本 `>=0.5.0 <1.0.0`。其中通过请求头签名的 `native` 命令使用 `CTYUN_STORAGE_ENDPOINT`（不包含桶名或路径的 HTTPS 服务端点）、`CTYUN_STORAGE_AK` 和 `CTYUN_STORAGE_SK`。`CTYUN_STORAGE_REGION` 应设置为服务文档中的签名区域，不能使用 OpenAPI 资源池 ID。`CTYUN_STORAGE_SIGNATURE_VERSION` 默认为 `v4`；使用 `v2` 时不需要签名区域。请求头签名使用 `CTYUN_STORAGE_SECURITY_TOKEN` 传递临时凭证，不复用 EOP 配置档案的端点或凭证。媒体存储将桶名放在路径中，经典版将桶名放在主机名中。

经典版存储的 `native statistics` 使用 `s3` 签名服务和 `cn-mg` 等签名区域；`native tracking` 使用 `cloudtrail`，`native iam` 使用 `sts`，二者使用 `cn` 等签名区域。请根据对应服务文档选择端点和签名区域。操作跟踪和 IAM 要求使用 V4。IAM 的 `--tags` 接受由 `Key`/`Value` 对象组成的 JSON 数组，`--tag-keys` 接受 JSON 字符串数组。表单值由 CLI 执行 URL 编码，输入时无需预先编码。

## 策略上传

使用 `ctyun media-storage object post {bucket_name}` 时，提供 `--key` 和 `--file`。需要存储签名时，通过 `--storage-access-key` 指定访问密钥 ID，通过 `--policy` 传入 Base64 编码的 JSON 对象。可以用 `--signature` 传入已有签名，也可以设置 `CTYUN_STORAGE_SK` 让 CLI 在本地计算 V2 签名。三个存储签名选项均省略时，服务将存储上传视为匿名请求，只有允许公开写入的桶才接受此类请求。网关请求仍使用常规的 [EOP 凭据](configuration.md)。原生 `object post` 使用相同的 V2 策略输入，路由只需 `CTYUN_STORAGE_ENDPOINT`；`CTYUN_STORAGE_SIGNATURE_VERSION` 不改变其策略算法。通过 `--security-token` 传递临时存储凭证。两条命令都接受 `--success-action-status` 和 `--success-action-redirect`。CLI 在网关表单中发送连字符形式的字段名，在原生表单中发送 `success_action_status` 和 `success_action_redirect`。成功重定向只返回目标地址，不会访问该地址。

## 文件输入与输出

仅在声明文件输入的命令上使用 `--file` 或 `--document-file`；普通 `@` 开头的值不会被当作文件读取。原生上传提供 `--content-type`，支持元数据的命令通过 `--metadata` 接收元数据名称与字符串值组成的 JSON 对象。上传需要临时磁盘空间。XML 输入文档和按显式传输策略解码的响应上限为 16 MiB。二进制下载以流式方式写入目标。命令帮助会列出可用输出格式：`--output raw` 原样输出响应体，不能与表格控制或等待器混用；下载二进制内容时，用 `--output raw` 写入标准输出，或用 `--output-file <path>` 写入文件。需要替换已有常规文件时，可在文件输出选项后加上 `--overwrite`。

## 分页与等待器

媒体存储原生列表的分页由用户显式控制：通过 `--output raw` 或 `--output json` 读取后续页标记，再传入下一次请求。对象可用等待器将有效的解冻对象和非归档对象视为可用；操作跟踪等待器检查跟踪是否开启或关闭，不代表日志投递正常。

## 列出对象并下载文件

运行 `ctyun plugin install media-storage --channel beta` 安装媒体存储插件，然后按上文设置原生端点、存储凭证和签名区域。将 `{bucket}` 换成有权读取的桶名，`{object_name}` 换成已有对象的键。值中含空格时请加引号。

```sh
ctyun media-storage native object list {bucket} --max-keys 10
ctyun media-storage native object show {bucket} {object_name} --output-file download.bin
```

第一条命令请求一页对象列表，最多返回十个键。第二条将对象内容写入本地文件。请选择尚未使用的文件名，明确需要替换已有文件时才加上 `--overwrite`。
