# Changelog

## 0.1.0-beta.1 - 2026-09-23

### Added

- Introduced the Classic Object Storage plugin with 219 commands: 106 CTyun OpenAPI operations and 113 native bucket, object, statistics, tracking, and IAM operations.
- Added localised command help and table labels, operation-specific request encodings and HTTP response contracts, XML and header-only output, and documentation-derived fixtures with recorded corrections.
- Added a separate `native` command group with independent storage credentials and endpoints, V2/V4 signing, and exact object-key paths. Tracking and IAM require V4 signing; IAM forms support indexed tag lists. Live interoperability remains unverified.
- Added logging enabled/disabled waiters bound to the tracking status query.
- Set the initial required core range to `>=0.5.0 <1.0.0`.
