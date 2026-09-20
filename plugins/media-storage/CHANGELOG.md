# Changelog

## Unreleased

### Added

- Added 124 commands (74 CTyun OpenAPI and 50 native bucket/object APIs) with explicit HTTP response contracts, documentation-derived fixtures with explicit correction provenance, and localized command help and table labels.
- Added namespace-aware XML output and HTTP status/header output; request encodings follow the captured operation contracts.
- Covers all 74 captured operations, including POST upload with separately supplied V2 policy signatures or local signing using CTYUN_STORAGE_SK, metadata fields, and temporary storage tokens. Corrections, inferred redirect behaviour, and synthetic fixtures are identified in the catalog inventory.
- Native commands use a separate `native` command group, storage credentials and endpoints, V2/V4 signing, exact object-key paths, and documented native response contracts; live interoperability remains unverified.
- Requires core `>=0.5.0 <1.0.0`.
