# Changelog

## 0.1.0 - 2026-07-04

### Added

- Global `--version` and `-v` flags alongside the existing `ctyun version`
  command.
- Dev-only OpenAPI harvest, diff, generate, review, and promotion tooling under
  `tools/openapi`, backed by normalized product source catalogs.
- OpenAPI source fingerprints, tracked source/baseline evidence, and ignored
  reproducible review outputs for reviewed or curated plugin metadata.
- Repo-local fixture smoke checks for real bundled plugin metadata in
  `tools/plugincheck`.
- Plugin storefront, install, reinstall, update, list, and search workflow
  coverage for hosted metadata and development bundled plugins.

### Changed

- Refined core and plugin help output so usage lines, argument sections, required
  options, option value placeholders, allowed values, defaults, and plugin
  discovery sections use one consistent format.
- Updated release packaging to merge rebuilt core and plugin indexes into
  existing output instead of replacing unrelated channels or plugin artifacts.
- Improved README onboarding, plugin workflow, versioning, and release examples
  for the stable release line.

### Fixed

- Fixed plugin compatibility checks to use full SemVer prerelease precedence, so
  a `0.1.0-alpha.1` core no longer accepts plugin constraints such as
  `>=0.1.0 <1.0.0`.
- Allowed `0.1.0-dev` source builds to satisfy the matching stable base
  compatibility range while keeping prerelease ordering intact.
- Fixed installer and update channel selection to prefer stable releases before
  beta and alpha when no channel is pinned.
- Improved table rendering for array and object values so multi-value fields are
  readable instead of Go's default space-separated formatting.
- Improved localized diagnostics for plugin metadata, registry, distribution, and
  API error paths.

## 0.1.0-alpha.1 - 2026-06-21

Initial alpha release.

### Added

- Native `ctyun` CLI with localized help, table output by default, and JSON fallback.
- Config and profile commands with environment-variable credential precedence.
- Metadata-driven plugin command loading with bundled ECS and Region plugins.
- Offline fixture mode for development and safe command-shape verification.
- Signed core self-upgrade metadata, hosted mirror selection, and native installer scripts.
- Signed hosted plugin registry metadata with GitHub and Gitee mirror support.
- Release packaging for core binaries, installer scripts, plugin archives, and signed indexes.
