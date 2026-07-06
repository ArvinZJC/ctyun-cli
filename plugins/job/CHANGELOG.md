# Changelog

## Unreleased

### Added

- Generated Job plugin metadata from the official CTyun OpenAPI documentation
  for `GET /v4/job/info`.
- Tracked OpenAPI `source.json` and promoted `baseline.json` evidence for Job.
- Command, API, table, fixture, and help metadata for the official Job API
  whose URI starts with `/v4/job/`.

### Changed

- `job info` now keeps `{job_id}` as the task resource argument and accepts
  optional `--region` as an override for the profile-scoped `regionID`.
- `job info` examples now use the captured official `jobID` value from upstream
  example data instead of the `{job_id}` placeholder.
- Removed the raw `fields` object selector from table help while keeping the
  readable `task_name` projection.
