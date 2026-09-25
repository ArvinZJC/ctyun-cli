# Changelog

## Unreleased

### Added

- Add 306 commands from the published Cloud Container Engine API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 296 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 4 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for the declared response contracts and metadata features.

### Known limitations

- `cce v1-cce node-pool custom create {cluster_name}`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cce v1-ccse node-pool custom create {cluster_name}`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cce kubernetes node-autoscaler list {cluster_id}`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cce kubernetes node-autoscaler show {cluster_id} {name}`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cce cluster upgrade-check {cluster_id}`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cce v1-ccse cluster upgrade-check {cluster_name}`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cce kubernetes snapshot-class show {cluster_id} {api_version} {volume_snapshot_class_name}`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cce v1-cce cluster upgrade-check {cluster_name}`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cce cluster specification task show {cluster_id}`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cce kubernetes pod update {cluster_id} {namespace_name} {pod_name}`: The official Pod update example is a statusCode 1001 / CCSE_1001 validation failure, not a successful response.

See `openapi-catalogs/cce/coverage.json` for the waiter assessment and upstream evidence limitations.
