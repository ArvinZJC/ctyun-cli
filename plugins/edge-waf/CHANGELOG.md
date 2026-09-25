# Changelog

## Unreleased

### Added

- Add 68 commands from the published Web Application Firewall (Edge Cloud Edition) API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 68 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 2 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

See `openapi-catalogs/edge-waf/coverage.json` for the waiter assessment and upstream evidence limitations.
