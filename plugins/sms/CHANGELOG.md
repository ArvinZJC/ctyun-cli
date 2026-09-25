# Changelog

## Unreleased

### Added

- Add 24 commands from the published Cloud Messaging - SMS API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 24 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 4 bounded waiters using documented terminal states from resource detail queries.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

See `openapi-catalogs/sms/coverage.json` for the waiter assessment and upstream evidence limitations.
