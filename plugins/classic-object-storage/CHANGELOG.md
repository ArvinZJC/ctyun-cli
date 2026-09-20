# Changelog

## Unreleased

### Added

- Added 150 commands (106 CTyun OpenAPI and 44 native bucket/object APIs) with explicit HTTP response contracts, documentation-derived fixtures with explicit correction provenance, and localized command help and table labels.
- Added namespace-aware XML output and HTTP status/header output; request encodings follow the captured operation contracts.
- Covers all 106 captured OpenAPI operations; repaired documentation examples and companion references are recorded in the catalog inventory.
- Native commands use a separate `native` command group, storage credentials and endpoints, V2/V4 signing, exact object-key paths, and documented native response contracts; live interoperability remains unverified.
- Requires core `>=0.5.0 <1.0.0`.
