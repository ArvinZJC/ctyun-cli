# Changelog

## Unreleased

### Added

- Add 47 commands from the published VPC Endpoints API catalog, including endpoint tags and object storage endpoint policies, with Chinese and English help, explicit request bindings and response success checks.
- Include 47 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add one bounded waiter using documented terminal states from resource detail queries.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Fixed

- Accept endpoint order processing status 900 only when the response contains `returnObj.masterOrderID`; ordinary status 900 failures remain errors, and documented status 800 remains successful.

See `openapi-catalogs/vpce/coverage.json` for the waiter assessment and upstream evidence limitations.
