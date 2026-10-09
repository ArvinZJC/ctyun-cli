# Changelog

## Unreleased

### Added

- Add 16 commands from the published Billing API catalog, with Chinese and English help, explicit request bindings and current response success checks, clear billing-date and payment-type labels, and optional account and user IDs retained from the usage-period request example.
- Include 16 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

See `openapi-catalogs/billing/coverage.json` for the waiter assessment and upstream evidence limitations.
