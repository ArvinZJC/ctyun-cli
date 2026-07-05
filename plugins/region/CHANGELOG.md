# Changelog

## 0.2.0 - 2026-07-05

### Added

- Customer resource and quota summary commands from the official resource-pool
  API documentation.

### Changed

- Expanded product, resource, and quota table coverage with top-level returned
  objects, keeping default field subsets for easier single-resource scanning.
- Refined single-resource table defaults to concise drilled summary fields and
  disambiguated resource summary labels such as Cloud Backup (CBR) and Volume
  Backup (VBS).
- Demand-check metadata now treats only guarded `900` responses with
  `returnObj.satisfied` as accepted, keeps EIP amount optional, and documents
  ECS/EBS conditional options in help.
- Promoted metadata quality to `curated`.
- Raised the required core range to `>=0.2.0 <1.0.0` for the new metadata and
  output behaviours.

## 0.1.0 - 2026-07-04

### Added

- Region summary and zone-list command metadata.
- Region product-info and demand-check command metadata.
- Additional table columns from the official resource-pool responses.

### Changed

- Promoted metadata quality to `reviewed` and channel to `stable`.
- Updated the required core range to `>=0.1.0 <1.0.0`.

## 0.1.0-alpha.1 - 2026-06-21

### Added

- Region list command metadata.
- Table definitions, localized help text, and offline fixtures.
- Live CTyun API mapping for region queries.
