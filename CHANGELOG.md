# Changelog

## 0.2.0 - 2026-07-05

### Added

- Plugin metadata can declare vertical table layouts, default column subsets,
  conditional option requirements, and guarded CTyun `900` accepted responses.

### Changed

- Single-object plugin tables can render as localized field/value rows, with
  default column subsets used when metadata declares them.
- Table selector help and diagnostics now distinguish horizontal columns from
  vertical fields, list selectors one per line, show declared defaults as
  separate markers, and keep stable keys available for `--cols`, `--filter`,
  and `--sort`.
- Object-valued table cells now use a consistent nested `key=value` format.
- Plugin discovery can list or search all registry channels with
  `--channel all`, while install and update commands validate that a concrete
  channel was selected.
- Plugin command parameter errors now omit internal command IDs and focus on the
  actionable flag or condition.
- Plugin help annotates conditionally required options while keeping them
  optional in usage until their condition applies.
- Development builds still discover bundled source-tree plugins, while release
  builds now load only installed plugin bundles.

### Fixed

- Chinese plugin command group help now avoids template-introduced spaces such
  as `管理 弹性云主机 instance 命令`.
- Release packaging now builds with `-trimpath` so binary artifacts do not
  embed local checkout paths.

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

- Fixed core update and release comparisons to use full SemVer prerelease
  precedence, so stable `0.1.0` is treated as newer than `0.1.0-alpha.1`.
- Fixed plugin compatibility checks to use the same SemVer prerelease
  precedence, so a `0.1.0-alpha.1` core no longer accepts plugin constraints
  such as `>=0.1.0 <1.0.0`.
- Allowed `0.1.0-dev` source builds to satisfy the matching stable base
  compatibility range while keeping prerelease ordering intact.
- Fixed installer and update channel selection to prefer stable releases before
  beta and alpha when no channel is pinned.
- Improved table rendering for array and object values so multi-value fields are
  readable instead of Go's default space-separated formatting.
- Improved localized diagnostics for plugin metadata, registry, distribution, and
  API error paths.

## 0.1.0-alpha.1 - 2026-06-21

### Added

- Native `ctyun` CLI with localized help, table output by default, and JSON fallback.
- Config and profile commands with environment-variable credential precedence.
- Metadata-driven plugin command loading with bundled ECS and Region plugins.
- Offline fixture mode for development and safe command-shape verification.
- Signed core self-upgrade metadata, hosted mirror selection, and native installer scripts.
- Signed hosted plugin registry metadata with GitHub and Gitee mirror support.
- Release packaging for core binaries, installer scripts, plugin archives, and signed indexes.
