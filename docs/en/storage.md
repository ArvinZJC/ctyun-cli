# Storage credentials and file transfers

[Project home](../../README-EN.md) · [简体中文](../zh-CN/storage.md)

## Native storage credentials

The `media-storage` and `classic-object-storage` plugins require core `>=0.5.0 <1.0.0`. Their header-signed `native` commands use `CTYUN_STORAGE_ENDPOINT` (an HTTPS service endpoint without a bucket or path), `CTYUN_STORAGE_AK`, and `CTYUN_STORAGE_SK`. Set `CTYUN_STORAGE_REGION` to the signing region in the service documentation, not an OpenAPI resource-pool ID. `CTYUN_STORAGE_SIGNATURE_VERSION` defaults to `v4`; `v2` does not require a signing region. Use `CTYUN_STORAGE_SECURITY_TOKEN` for temporary credentials in header-signed requests. EOP profile endpoints and credentials are not reused. Media uses path-style bucket addressing; classic uses virtual-host addressing.

For classic storage, `native statistics` uses signing service `s3` and a region such as `cn-mg`; `native tracking` uses `cloudtrail`, and `native iam` uses `sts`, both with a region such as `cn`. Select the endpoint and signing region from the corresponding service documentation. Tracking and IAM require V4. IAM `--tags` accepts a JSON array of `Key`/`Value` objects, and `--tag-keys` accepts a JSON string array. Supply form values unencoded; the CLI URL-encodes them.

## Policy-based uploads

For `ctyun media-storage object post {bucket_name}`, provide `--key` and `--file`. Storage authentication also requires `--storage-access-key` and a Base64-encoded JSON object in `--policy`; supply `--signature` or set `CTYUN_STORAGE_SK` for local V2 signing. If you omit all three storage signing options, the service treats the storage upload as anonymous. This requires a bucket that permits public writes. Gateway requests still use the normal [EOP credentials](configuration.md). Native `object post` uses the same V2 policy inputs and only `CTYUN_STORAGE_ENDPOINT` for routing; `CTYUN_STORAGE_SIGNATURE_VERSION` does not change its policy algorithm. Pass temporary storage credentials with `--security-token`. Both commands accept `--success-action-status` and `--success-action-redirect`. The CLI sends hyphenated names in the gateway form and `success_action_status` and `success_action_redirect` in the native form. Successful redirects are reported without being followed.

## File input and output

Use `--file` or `--document-file` only on commands that declare file inputs; ordinary values beginning with `@` are not read as files. Native uploads offer `--content-type` and, where supported, `--metadata` as a JSON map of metadata names to string values. Uploads require temporary disk space. The CLI limits XML input documents and responses decoded under an explicit transport policy to 16 MiB. Binary downloads stream to their destination. Command help lists available output formats: `--output raw` writes exact response bytes and cannot be combined with table controls or waiters; for binary downloads, choose either `--output raw` for stdout or `--output-file <path>` for a file. Use `--overwrite` with the file option to replace an existing regular file.

## Pagination and waiters

Media native pagination is explicit: read continuation markers with `--output raw` or `--output json` and pass them to the next request. The object availability waiter recognises valid restored objects and non-archived objects. Tracking waiters check whether logging is enabled or disabled, not whether log delivery is healthy.

## List objects and download one file

Install the Media Storage plugin with `ctyun plugin install media-storage --channel beta`. Set the native endpoint, storage credentials and signing region described above. Replace `{bucket}` and `{object_name}` with a bucket you can read and an existing object key. Quote either value if it contains spaces.

```sh
ctyun media-storage native object list {bucket} --max-keys 10
ctyun media-storage native object show {bucket} {object_name} --output-file download.bin
```

The first command requests a page of up to ten keys. The second writes the object bytes to a local file. Choose an unused filename, or pass `--overwrite` when you intend to replace an existing file.
