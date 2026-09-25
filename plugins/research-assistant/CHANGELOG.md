# Changelog

## Unreleased

### Added

- Add 46 commands from the published Research Assistant API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 43 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 3 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Known limitations

- `research-assistant queue create`: The published example omits the statusCode/returnObj envelope required by the response schema; no enclosing success fixture is fabricated.
- `research-assistant resource usage show`: The published example omits the statusCode/returnObj envelope required by the response schema; no enclosing success fixture is fabricated.
- `research-assistant resource instance-quota show`: The published example omits the statusCode/returnObj envelope required by the response schema; no enclosing success fixture is fabricated.

See `openapi-catalogs/research-assistant/coverage.json` for the waiter assessment and upstream evidence limitations.
