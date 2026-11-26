# Changelog

## 0.1.0-beta.2 - 2026-11-26

### Fixed

- Label the `os_type` column in `ctyun server-security vulnerability last-scan show` as “Vulnerability Types” in English and “漏洞类型” in Chinese. Its values include `WEB_CMS` and `EMERGENCY_VUL` as well as `LINUX` and `WINDOWS`.

## 0.1.0-beta.1 - 2026-10-10

### Added

- Add 154 commands from the published Server Security (Native Edition) API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 154 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 9 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for the declared response contracts and metadata features.

See `openapi-catalogs/server-security/coverage.json` for the waiter assessment and upstream evidence limitations.
