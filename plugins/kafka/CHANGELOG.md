# Changelog

## Unreleased

### Added

- Add 124 commands from the published Kafka API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 118 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 2 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- `kafka instance configuration legacy-update`: The published response is a schema illustration: statusCode and other values contain type descriptions rather than a real success response. The explicit 800 success contract is supported without fabricating a fixture.
- `kafka tag create`: The published response is a schema illustration: statusCode and other values contain type descriptions rather than a real success response. The explicit 800 success contract is supported without fabricating a fixture.
- `kafka tag resource bind`: The published response is a schema illustration: statusCode and other values contain type descriptions rather than a real success response. The explicit 800 success contract is supported without fabricating a fixture.
- `kafka tag resource unbind`: The published response is a schema illustration: statusCode and other values contain type descriptions rather than a real success response. The explicit 800 success contract is supported without fabricating a fixture.
- `kafka fault-drill cancel`: No response example is published; the response schema explicitly defines success 800 and failure 900.
- `kafka v2 instance list`: The published response is a schema illustration: statusCode and other values contain type descriptions rather than a real success response. The explicit 800 success contract is supported without fabricating a fixture.

See `openapi-catalogs/kafka/coverage.json` for the waiter assessment and upstream evidence limitations.
