# Changelog

## Unreleased

### Added

- Add 207 commands from the published Type II Redis API catalog, covering instance lifecycle, accounts, networking, backups, configuration, monitoring, diagnostics, data migration, tags and command auditing; retain published historical APIs under `redis legacy`.
- Include Chinese and English help, captured response fixtures, explicit API scopes and thirteen bounded waiters for instances, nodes, migration tasks, background tasks, flashback, parameter changes and exact-name backup or recovery completion.
- Require confirmation for mutations, including GET-based key analysis task creation, disable retries for state-changing operations, and validate the documented conditional inputs for prepaid creation, automatic renewal and deployment settings.
- Preserve all five tag groups in the offline tag-list fixture by repairing missing delimiters against the documented schema, with the original example and correction recorded in supplemental evidence.
- Require core `>=0.5.1 <1.0.0` for response contracts, waiter selection and independent request bindings.

### Fixed

- Expose the legacy resize precheck as `redis legacy instance resize-check` so the `resize` command cannot shadow it.

- Correct response labels against operation-specific meanings, including tag resource counts, migration conflicts, export formats, key-analysis times and eviction policy.

### Known limitations

- The separate Redis catalog labeled Type I (`vid=75`) is not captured. These product-specific catalog labels are distinct from the project's self-operated versus joint-operation node classification.
- Additional collection waiters require documented unique task inputs or more state evidence; see `openapi-catalogs/redis/coverage.json`.
