# Changelog

## 0.1.0-beta.2 - 2026-11-26

### Fixed

- In `ctyun whole-site-acceleration function usage show`, use `start_time_example` and `end_time_example` to select the timestamps whose field names have trailing spaces in the published response example. The `start_time` and `end_time` selectors still select the fields defined in the response schema.
- Remove the deprecation warning when creating a domain with `ctyun whole-site-acceleration domain create --action 1`. Only modification with `--action 2` is scheduled for shutdown. Use `ctyun whole-site-acceleration domain update` to modify a domain.

## 0.1.0-beta.1 - 2026-10-10

### Added

- Add 96 commands from the published Whole Site Acceleration API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 94 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 3 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Known limitations

- `whole-site-acceleration report-subscription update`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `whole-site-acceleration block area list`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/whole-site-acceleration/coverage.json` for the waiter assessment and upstream evidence limitations.
