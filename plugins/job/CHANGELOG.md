# Changelog

## 0.1.1 - 2026-07-21

### Changed

- Clarified the localized positional-argument label as an asynchronous task ID.

## 0.1.0 - 2026-07-11

### Added

- Curated Job plugin metadata from the official CTyun OpenAPI documentation for `GET /v4/job/info`.
- Tracked OpenAPI `source.json` and promoted `baseline.json` evidence for Job.
- Command, API, table, fixture, and help metadata for the official Job API whose URI starts with `/v4/job/`.
- `job info` keeps `{job_id}` as the task resource argument and accepts optional `--region` as an override for the profile-scoped `regionID`.
- `job info` examples use the captured official `jobID` value from upstream example data.
- A readable `task_name` table projection without exposing the raw `fields` object selector in table help.
- Metadata omits generated `900` accepted-status rules based on error-envelope fields so API failures surface as API status errors.
