# Changelog

## Unreleased

### Added

- Add 37 commands from the published Elastic Container Instances API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 36 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- `eci container-group price show`: The published example contains only priceInfo and omits the documented statusCode/returnObj envelope. Keep the documented statusCode=200 contract; do not fabricate the missing success envelope.

See `openapi-catalogs/eci/coverage.json` for the waiter assessment and upstream evidence limitations.
