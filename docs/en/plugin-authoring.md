# Authoring and reviewing plugins

[Project home](../../README-EN.md) · [简体中文](../zh-CN/plugin-authoring.md)

## Trace a command through its bundle

Use `ctyun region list --name 保定` to see how a command reaches the API. To add another product command, follow the same links between metadata files.

| File in the Region bundle | What the command uses |
|---|---|
| [plugin.json](../../plugins/region/plugin.json) | Declares the plugin name, release channel, quality, supported core range, API endpoint, URI scope and fingerprint of the promoted source |
| [commands.json](../../plugins/region/commands.json) | Binds the visible words `region list` to an operation and table, maps `--name` to parameter `regionName` and names its fixture |
| [apis.json](../../plugins/region/apis.json) | Declares a retryable `GET /v4/region/list-regions`; the query field `regionName` receives `$param.regionName` |
| [tables.json](../../plugins/region/tables.json) | Reads `returnObj.regionList`. Each row’s `regionID` becomes the stable column `region_id`, displayed as “Region ID” in English |
| [i18n](../../plugins/region/i18n) | Supplies descriptions of the command and its parameter in each supported language |
| [fixture](../../plugins/region/fixtures/region-list.json) | Supplies the recorded envelope and rows for offline verification |
| [waiters.json](../../plugins/region/waiters.json) | Holds polling definitions for commands with documented terminal states. Region has none |

From the repository root, run the same lookup in table and JSON modes. In the fixture table, the parameter selects the matching row. JSON exposes the recorded envelope, including the other row. For a live call, the service receives the query parameter.

```sh
go run ./cmd/ctyun plugin lint ./plugins/region
go run ./cmd/ctyun region list --name 保定 --offline
go run ./cmd/ctyun region list --name 保定 --offline --output json
```

For a path argument, bind `$arg.<name>`; the core escapes each substituted value as one path segment. For an operation that takes its region from the profile, expose the optional `--region` override. Keep request keys and payload values as published. Choose the table key and localised labels according to what the field means in that operation. Mark state-changing commands that require confirmation in the command metadata.

## OpenAPI catalogue workflow

The OpenAPI catalogue pipeline is a developer tool. It is not exposed as a user command and is not included in core or plugin release artefacts. It starts from normalised JSON input and stores upstream evidence in `openapi-catalogs/<name>/source.json`:

```sh
go run ./tools/openapi harvest <name> --input path/to/normalized-source.json
go run ./tools/openapi normalize-labels <name>
go run ./tools/openapi diff <name>
go run ./tools/openapi generate <name>
go run ./tools/openapi review <name>
```

The [product coverage inventory](../../openapi-catalogs/product-coverage.json) compares promoted commands with the official product catalogue, including cross-product references, reviewed equivalent APIs, excluded APIs, and confirmed documentation blockers. Equivalent coverage uses the target product endpoint and response contract; it does not claim support for the older catalogue’s response shape. The [official API review](../../openapi-catalogs/official-review.json) records the latest full-plugin check, evidence-backed updates, retained documentation gaps, and endpoint blockers.

Product-level records detail API gaps, source corrections or waiter limitations for [EIP](../../openapi-catalogs/eip/coverage.json), [PostgreSQL](../../openapi-catalogs/rds-postgresql/coverage.json), [Redis](../../openapi-catalogs/redis/coverage.json) and [MongoDB](../../openapi-catalogs/mongodb/coverage.json).

Update plugins in four stages: prepare the evidence, review the generated draft, check lifecycle waiters, then promote the accepted result.

### 1. Update the source evidence

Keep these files under `openapi-catalogs/<name>/`:

| File                                | Purpose                                                       | Git policy                     |
|-------------------------------------|---------------------------------------------------------------|--------------------------------|
| `source.json`                       | Current upstream evidence, API scope and metadata to generate | Track                          |
| `baseline.json`                     | Last accepted snapshot used to validate the promoted plugin   | Track; updated by promotion    |
| `draft/`, `changes.md`, `review.md` | Generated bundle, change report and review results            | Ignore; regenerate when needed |

A newer `source.json` may differ from `baseline.json` while changes await review. Until promotion, validate the existing plugin's source fingerprint and API scope against `baseline.json`. Keep routine history in git.

Start with the CTyun product and service index. For each service, enumerate every entry in `ctapiDocumOut/queryByserviceId/{serviceId}` and fetch each menu. Record its upstream label, ID, URL and operation count. Compare the promoted API IDs with the union of relevant menus, including secondary catalogues and operations owned by another product. Record a failed fetch as unverified. Mark an operation unlisted only after every relevant menu has been checked. Record deprecation separately, using the published notice and its exact scope.

Use the documentation URL’s `sid` for `product.ctyun_product_id` and its `vid`, when present, for `product.source_revision`. A revision identifier is a provenance hint. Compare the captured request and response contracts even when that identifier stays the same.

When updating `source.json`:

- Declare the supported API URI ranges in `product.api_scope`. Generation, review and promotion must stay within those ranges.
- Capture complete request examples in `request_example` and individual parameter values in `example`. If upstream provides no usable example, confirm that gap during review and record `example_unavailable`.
- Preserve the evidence for recommended alternative APIs. A recommendation without deprecation or shutdown wording must not become a deprecation warning.

### 2. Generate and review the draft

Use `diff`, `generate` and `review` to rebuild the local outputs. `normalize-labels` can repair shared technical casing and reviewed phrases in `source.json`; uncertain labels stay unchanged and must be resolved before review can pass.

Run the draft and compare its behaviour with the source catalogue. Check the API contract, command bindings, help, examples and fixtures:

- Ordinary JSON fixtures and HTTP response fixtures must match captured source evidence. Missing or changed fixture data blocks review.
- Keep table row selectors, column keys and paths, and column deprecation metadata consistent with the source catalogue. You can change the layout and default columns during review.
- Examples must add something beyond the Usage line: concrete arguments, meaningful options, structured values or other behaviour. Examples that only repeat the command path, including unresolved path placeholders, are rejected.
- Check that the draft includes a command description in each supported language. Rewrite mechanically assembled English descriptions. Check examples for missing required options, undeclared options and values that do not match their parameter types.
- Generate recommendation help only when the target resolves to a tracked, promoted CLI command. Otherwise, retain the unresolved evidence in `source.json`. Cross-plugin references are soft dependencies at load time, but release checks require an exact, non-deprecated target and reject recommendation cycles.

If an API has a clear response contract but no usable successful response example, set `fixture_unavailable` with the reason. Keep the command, response validation and original evidence; do not invent a success fixture. The command cannot use `--offline` or `--fixture`, and the missing example cannot serve as waiter evidence. For downloads, declare the documented success `media_type` on the response variant so an HTTP-200 error envelope is not saved as a file.

### 3. Check lifecycle waiters

At each review, assess whether asynchronous operations have a safe retrieval command with documented terminal states. Bind a waiter only to a non-dangerous, explicitly retryable retrieval operation. Record missing state evidence or resource-identifying inputs as gaps instead of guessing when an operation is complete.

Keep waiter definitions and their upstream evidence in the catalogue's `waiters` map. Review rejects waiter definitions that differ from the source catalogue. Promotion preserves accepted definitions. If you change the definitions, regenerate and review the draft before promotion. Test the real bundle offline for response shape, terminal outcomes, and command-specific help and completion.

<details>
<summary>Waiter metadata reference</summary>

| Field                              | Requirement                                                                                                                              |
|------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| `commands`                         | Explicit bindings to exact command IDs                                                                                                   |
| `path` or `xml_path`               | JSON state path, or an absolute namespace-aware XML path selecting exactly one scalar element; XML paths cannot use collection selectors |
| `success`, `failure`               | Documented terminal values; leave both failure fields empty when no failure condition is documented                                              |
| `success_values`, `failure_values` | Additional documented terminal values when needed                                                                                        |
| `max_attempts`, `interval_seconds` | Positive attempt limit and polling interval in seconds                                                                                   |
| `evidence`                         | Source and meaning of the documented states                                                                                              |

For collection responses, add a `selector`: `path` locates the collection, `key` identifies each row, and `value` references an existing `$arg.<name>` or scalar string/integer `$param.<name>`. The state path is relative to the selected row. Check every captured row, preserve exact numeric identities, and keep null or unknown states pending. If a unique identity input is unavailable, record the blocker.

</details>

### 4. Promote the accepted draft

Before promotion, confirm that review passes and the declared quality (`generated`, `reviewed` or `curated`) accurately describes the plugin.

Generation derives `source_fingerprint` from `source.json`. For existing plugins, it preserves the promoted version, channel, quality and core compatibility range, raising the minimum core version when required:

| Metadata feature | Minimum core version |
|------------------|----------------------|
| Basic generated bundle | `0.3.1` |
| Non-string request-body parameters | `0.4.0` |
| Waiters, explicit transport policies, native storage, downloads or XML response tables | `0.5.0` |
| Independent request-header, query and body bindings | `0.5.1` |

Regeneration must not lower an existing compatibility requirement or reset the plugin version, channel or quality. Promotion updates plugin metadata and advances `baseline.json` to the accepted source:

```sh
go run ./tools/openapi promote <name>
```
