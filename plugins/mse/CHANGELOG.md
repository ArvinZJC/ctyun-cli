# Changelog

## Unreleased

### Added

- Add 243 commands from the published Microservice Engine API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 232 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 11 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for the declared response contracts and metadata features.

### Known limitations

- `mse governance circuit-breaker-rule list`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `mse governance tag-route list`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `mse instance list`: The captured response uses an absent, zero or inconsistent status field, contradicting the published statusCode 2000 schema; it is not a successful offline fixture.
- `mse gateway extension update`: The captured response uses an absent, zero or inconsistent status field, contradicting the published statusCode 2000 schema; it is not a successful offline fixture.
- `mse instance node status`: The captured response uses an absent, zero or inconsistent status field, contradicting the published statusCode 2000 schema; it is not a successful offline fixture.
- `mse gateway domain create`: The captured response uses an absent, zero or inconsistent status field, contradicting the published statusCode 2000 schema; it is not a successful offline fixture.
- `mse gateway route rate-limit-policy update`: The captured response uses an absent, zero or inconsistent status field, contradicting the published statusCode 2000 schema; it is not a successful offline fixture.
- `mse gateway api-group response list`: The captured response uses an absent, zero or inconsistent status field, contradicting the published statusCode 2000 schema; it is not a successful offline fixture.
- `mse nacos credentials permission create`: The captured response uses an absent, zero or inconsistent status field, contradicting the published statusCode 2000 schema; it is not a successful offline fixture.
- `mse nacos credentials create`: The captured response uses an absent, zero or inconsistent status field, contradicting the published statusCode 2000 schema; it is not a successful offline fixture.
- `mse nacos credentials list`: The captured response uses an absent, zero or inconsistent status field, contradicting the published statusCode 2000 schema; it is not a successful offline fixture.

See `openapi-catalogs/mse/coverage.json` for the waiter assessment and upstream evidence limitations.
