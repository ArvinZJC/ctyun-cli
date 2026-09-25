# Changelog

## Unreleased

### Added

- Add 67 commands from the published Model Training and Inference API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 58 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 3 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- `model-training task event list`: The published log response is explicitly abbreviated with an omitted-content marker and is not valid complete JSON. The documented statusCode 200 envelope is supported without fabricating omitted records.
- `model-training task log list`: The published log response is explicitly abbreviated with an omitted-content marker and is not valid complete JSON. The documented statusCode 200 envelope is supported without fabricating omitted records.
- `model-training ide create`: The response schema explicitly documents statusCode 200 for success and 400 for failure, but no response example is published.
- `model-training ide name check`: The response schema explicitly documents statusCode 200 for success and 400 for failure, but no response example is published.
- `model-training ide start`: The response schema explicitly documents statusCode 200 for success and 400 for failure, but no response example is published.
- `model-training ide stop`: The response schema explicitly documents statusCode 200 for success and 400 for failure, but no response example is published.
- `model-training ide delete`: The response schema explicitly documents statusCode 200 for success and 400 for failure, but no response example is published.
- `model-training ide list`: The response schema explicitly documents statusCode 200 for success and 400 for failure, but no response example is published.
- `model-training ide show`: The response schema explicitly documents statusCode 200 for success and 400 for failure, but no response example is published.

See `openapi-catalogs/model-training/coverage.json` for the waiter assessment and upstream evidence limitations.
