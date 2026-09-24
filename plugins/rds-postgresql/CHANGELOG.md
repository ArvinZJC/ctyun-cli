# Changelog

## Unreleased

### Added

- Add confirmed, non-retryable billing-conversion commands for pay-as-you-go and prepaid subscriptions and recycle-bin instance recovery, retaining documented query/header inputs without fabricated offline fixtures; recovery retains the documented optional one-month default for prepaid instances.

## 0.1.0-beta.1 - 2026-09-23

### Added

- Add the PostgreSQL database plugin with 153 commands from the 159 APIs published in the first-class-node service, covering instances, databases, accounts, schemas, extensions, backups, security, monitoring, parameters, tags and orders.
- Include Chinese and English help, captured response fixtures, documented response envelopes, deprecated response fields and bounded instance/provisioning waiters.
- Require confirmation for mutations, including upstream GET operations that renew, unsubscribe, start or stop resources; validate documented database/recovery inputs and reject nested platform failures.

### Known limitations

- Six download/export APIs remain unpromoted after rechecking official documentation because no successful file HTTP representation distinguishes downloads from JSON business-error envelopes. Missing fixture bytes alone are not the blocker; see `openapi-catalogs/rds-postgresql/coverage.json`.
- Backup task polling requires a documented exact task identity input and is not inferred from the first collection row.
