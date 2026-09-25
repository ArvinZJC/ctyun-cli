# Changelog

## Unreleased

### Added

- Add 88 commands from the published Yunxiao Intelligent Computing Platform API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 64 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 7 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- Queue node specification listing has no published response contract and remains unavailable.
- Script content and task startup commands must be prepared using the upstream encoding or encryption requirements; the CLI does not encrypt supplied content.

- `yunxiao dataset show`: The only published response example is an authorization failure (statusCode 900). Success is explicitly documented as 800; no successful fixture is fabricated.
- `yunxiao queue show`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao queue workspace bind`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao mount list`: The only published response example is an authorization failure (statusCode 900). Success is explicitly documented as 800; no successful fixture is fabricated.
- `yunxiao mount node list`: The only published response example is an authorization failure (statusCode 900). Success is explicitly documented as 800; no successful fixture is fabricated.
- `yunxiao mount show`: The only published response example is an authorization failure (statusCode 900). Success is explicitly documented as 800; no successful fixture is fabricated.
- `yunxiao dataset list`: The only published response example is an authorization failure (statusCode 900). Success is explicitly documented as 800; no successful fixture is fabricated.
- `yunxiao queue quota show`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao node stop`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao node start`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao node reboot`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao image tag delete`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao image repository delete`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao image registry login show`: The only published response example is an authorization failure (statusCode 900). Success is explicitly documented as 800; no successful fixture is fabricated.
- `yunxiao image registry certificate show`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao node pod list`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao queue user list`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao queue user update`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao queue update`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao queue workspace unbind`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao node machine list`: The only published response example is an authorization failure (statusCode 900). Success is explicitly documented as 800; no successful fixture is fabricated.
- `yunxiao node lock`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao node show`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `yunxiao resource-group show`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/yunxiao/coverage.json` for the waiter assessment and upstream evidence limitations.
