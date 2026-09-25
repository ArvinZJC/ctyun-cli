# Changelog

## Unreleased

### Added

- Add 63 commands from the published Application Performance Monitoring API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 63 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 6 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Expose batch deletion through `--ids-file`, which sends the documented top-level JSON array without an object wrapper or numeric conversion.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

See `openapi-catalogs/apm/coverage.json` for the waiter assessment and upstream evidence limitations.
