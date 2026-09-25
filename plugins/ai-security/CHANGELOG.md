# Changelog

## Unreleased

### Added

- Add 5 commands from the published AI Security API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 4 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- Application authentication requires the documented application code, timestamp and precomputed HMAC signature in addition to EOP credentials.
- Watermark downloads accept the documented `application/octet-stream` representation; the upstream page does not identify its alternative file MIME types.

- `ai-security watermark download`: The official page documents a binary download but supplies no file bytes; JSON examples are error responses, not successful fixtures.

See `openapi-catalogs/ai-security/coverage.json` for the waiter assessment and upstream evidence limitations.
