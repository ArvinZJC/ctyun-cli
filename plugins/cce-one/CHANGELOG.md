# Changelog

## Unreleased

### Added

- Add 41 commands from the published CCE One API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 39 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 6 bounded waiters using documented terminal states from resource detail queries.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- `cce-one federation kubeconfig temporary show {federation_id}`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cce-one cluster kubeconfig temporary show {cluster_id}`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/cce-one/coverage.json` for the waiter assessment and upstream evidence limitations.
