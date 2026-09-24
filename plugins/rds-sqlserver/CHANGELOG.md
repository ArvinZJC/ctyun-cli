# Changelog

## Unreleased

### Added

- Add confirmed, non-retryable billing-conversion commands for pay-as-you-go and prepaid subscriptions, retaining documented query/header inputs without fabricated offline fixtures.

## 0.1.0-beta.1 - 2026-09-23

### Added

- Add the SQL Server plugin with 114 commands from the 114 APIs captured in the official first-class SQL Server documentation, covering instances, databases, accounts, backups, recovery, parameters, monitoring, logs, tasks, availability, and order queries.
- Include Simplified Chinese, British English, and American English help, captured response fixtures, and six bounded waiters for instance readiness, backup completion, recovery, object-storage recovery, tasks, and parameter-template application.
- Preserve header-based region resolution, typed request values, policy-specific ordinary recovery prerequisites with the recycle-bin exception, option enums, dangerous-operation confirmation, and explicit success checks for SQL Server's alternative response envelopes.

### Changed

- Require CTyun core 0.5.0 or later for explicit response contracts and identity-selected waiters.

### Known limitations

- The extended event report download follows the documented binary response contract; upstream supplies no response body, so this command has no offline fixture. Live interoperability remains unverified.
- Three malformed upstream JSON examples require recorded syntax repairs. Captured originals, response-shape contradictions, and the exact coverage boundary remain in the tracked OpenAPI catalog evidence.
