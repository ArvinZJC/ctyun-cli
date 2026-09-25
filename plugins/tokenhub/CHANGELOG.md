# Changelog

## Unreleased

### Added

- Add 19 commands from the published Xingchen TokenHub Operations Platform API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 18 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Known limitations

- `tokenhub order list`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/tokenhub/coverage.json` for the waiter assessment and upstream evidence limitations.
