# Changelog

## Unreleased

### Added

- Add 79 commands from the published Edge Security Acceleration Platform API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 79 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 7 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Known limitations

- Seven published APIs have no request or response schemas: ping probing, four resource-package operations, and image/text moderation. These remain unavailable pending an executable upstream contract.
- Batch domain, access-control and unsubscription responses expose individual outcomes; a successful outer envelope does not imply every item succeeded.

See `openapi-catalogs/edge-security-acceleration/coverage.json` for the waiter assessment and upstream evidence limitations.
