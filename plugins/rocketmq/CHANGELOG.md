# Changelog

## Unreleased

### Added

- Add 46 commands from the published RocketMQ API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 45 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add one bounded waiter using documented terminal states from resource detail queries.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- `rocketmq message trace show`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/rocketmq/coverage.json` for the waiter assessment and upstream evidence limitations.
