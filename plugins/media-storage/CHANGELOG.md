# Changelog

## Unreleased

### Added

- Introduced the Media Storage plugin with 124 commands: 74 CTyun OpenAPI operations and 50 native bucket and object operations.
- Added localized command help and table labels, operation-specific request encodings and HTTP response contracts, XML and header-only output, and documentation-derived fixtures with recorded corrections and synthetic examples.
- Added POST uploads with supplied V2 policy signatures or local signing using `CTYUN_STORAGE_SK`, metadata fields, temporary storage tokens, and explicitly modelled success redirects.
- Added a separate `native` command group with independent storage credentials and endpoints, V2/V4 signing, exact object-key paths, and SDK-documented listing pagination inputs. Live interoperability remains unverified.
- Added an XML object availability waiter bound to the restore-status query.
- Set the initial required core range to `>=0.5.0 <1.0.0`.
