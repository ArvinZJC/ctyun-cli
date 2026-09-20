# Changelog

## Unreleased

### Added

- Added 74 CTyun OpenAPI commands with explicit HTTP response contracts, documentation-derived fixtures with explicit correction provenance, and localized command help and table labels.
- Added namespace-aware XML output and HTTP status/header output; request encodings follow the captured operation contracts.
- Covers all 74 captured operations, including POST upload with separately supplied V2 policy signatures or local signing using CTYUN_STORAGE_SK, metadata fields, and temporary storage tokens. Corrections, inferred redirect behaviour, and synthetic fixtures are identified in the catalog inventory.
- Requires core `>=0.5.0 <1.0.0`.
