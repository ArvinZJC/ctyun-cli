# Changelog

## Unreleased

### Added

- Add 41 commands from the published AI Capability Platform API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 40 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- Speech synthesis remains unavailable because the published page has no request or response contract.

- `ai-capability speech recording recognize`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/ai-capability/coverage.json` for the waiter assessment and upstream evidence limitations.
