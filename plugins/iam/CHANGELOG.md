# Changelog

## Unreleased

### Added

- Add 87 commands from the published Identity and Access Management API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 79 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 2 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Known limitations

- Four MFA APIs lack request and response contracts, and identity-provider updates lack a consistent multipart upload contract; these five APIs are not exposed.

- `iam user password reset`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `iam delegation account list`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `iam user group remove`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `iam service permission list`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `iam service list`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `iam identity-provider user show`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `iam project group-policy show`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `iam identity-provider list`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/iam/coverage.json` for the waiter assessment and upstream evidence limitations.
