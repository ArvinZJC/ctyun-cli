# Changelog

## Unreleased

### Added

- Add 6 commands from the published Shared Traffic Packages API catalog, including traffic usage metrics, with Chinese and English help, explicit request bindings and response success checks.
- Include 6 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add one bounded waiter using documented terminal states from resource detail queries.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

See `openapi-catalogs/traffic-package/coverage.json` for the waiter assessment and upstream evidence limitations.
