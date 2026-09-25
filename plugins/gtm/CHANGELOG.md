# Changelog

## Unreleased

### Added

- Add 21 commands from the published Global Traffic Management API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 21 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 3 bounded waiters using documented terminal states from resource detail queries.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

See `openapi-catalogs/gtm/coverage.json` for the waiter assessment and upstream evidence limitations.
