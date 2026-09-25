# Changelog

## Unreleased

### Added

- Add 54 commands from the published ClickHouse API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 54 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- Two published APIs remain unimplemented because their documentation provides no response success contract: single-node restart (22426) and slow-query retrieval (22380).

See `openapi-catalogs/clickhouse/coverage.json` for the waiter assessment and upstream evidence limitations.
