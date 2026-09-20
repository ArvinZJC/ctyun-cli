# Changelog

## Unreleased

### Added

- Added 73 CTyun OpenAPI commands with explicit HTTP response contracts, documentation-derived fixtures with explicit correction provenance, and localized command help and table labels.
- Added namespace-aware XML output and HTTP status/header output; request encodings follow the captured operation contracts.
- Covers all captured operations except POST upload, which still requires separate policy signing; corrections and the synthetic download fixture are identified in the catalog inventory.
- Requires core `>=0.5.0 <1.0.0`.
