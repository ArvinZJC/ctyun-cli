# Changelog

## Unreleased

### Added

- Add 161 commands from the published API Gateway API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 156 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 8 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for the declared response contracts and metadata features.

### Known limitations

- `api-gateway order downgrade`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `api-gateway load-balancer listener update`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `api-gateway order on-demand update`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `api-gateway cloud-pc service activate`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `api-gateway project list`: The published response example describes joint-operation projects; the self-operated response schema is retained without fabricating a fixture.

See `openapi-catalogs/api-gateway/coverage.json` for the waiter assessment and upstream evidence limitations.
