# Changelog

## Unreleased

### Added

- Add 7 commands from the published Cloud Search Service API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 6 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Known limitations

- `cloud-search logstash list`: Published success example contains only the nested page result; response schema documents statusCode 200 and returnObj. No complete wire fixture is fabricated.

See `openapi-catalogs/cloud-search/coverage.json` for the waiter assessment and upstream evidence limitations.
