# Changelog

## 0.4.0 - 2026-07-17

### Added

- `ctyun config explain` now reports effective base settings and their winning sources without exposing credential or registry public-key material.
- `ctyun doctor local` now performs offline, read-only config and installed-plugin health checks, reports every independent finding, and uses a silent non-zero exit when any finding fails.
- `ctyun doctor network` now performs proxy-aware source, signed-index, and credential-free CTyun endpoint diagnostics with per-check status and timing.
- Plugin install, reinstall, update, and removal, plus applied core updates, now show terminal-aware progress on stderr and emit one localized completion summary on stdout; redirected and piped execution remains control-sequence free.
- Plugin command options can now declare typed boolean, numeric, array, map, and JSON values that are preserved when constructing API requests.
- Plugin command options can now declare integer-array values, preserving JSON integer semantics and large integer precision.
- Plugin metadata can now preserve recommendation-only API alternatives separately from deprecation and show a help-only visible command when the target plugin is loaded.
- Recommendation-only command guidance can now declare a localized applicability qualifier so help limits an alternative to the upstream-documented use case.

### Changed

- Interactive product-command and plugin-management tables now wrap localized text and long machine values to the detected terminal display width; redirected output retains its natural width.
- Command groups now render their help when invoked without a child, command options accept separated and inline values consistently, and invalid input uses generic unknown-option, unknown-command, unexpected-argument, or missing-argument diagnostics instead of command-specific usage errors.
- Development fixture options are now long-only product-command options placed after the complete command path; the former `-O` alias has been removed, and released builds no longer expose special development-option diagnostics.
- Network diagnostics now show transient interactive progress, emit a stable table or JSON report, and return non-zero only when a required capability is unavailable.
- Plugin install now skips plugins that are already installed without downloading or replacing them, including when a selected channel contains a lower version.
- Plugin reinstall now operates only on installed plugins and deliberately permits replacing the current version with the same or a lower version from the selected source and channel.
- Plugin update now installs only versions with strictly higher SemVer precedence.
- Multi-plugin operations preserve explicit argument order, use deterministic ordering for `--all`, continue independent work after individual failures, and return a combined failure after reporting the final result counts.
- Applied core updates now report download, verification, and installation as distinct progress phases; `--check` remains a non-progressing check.

### Fixed

- Development builds now allow hosted release checks with the normal `auto` default while rejecting core update installation before source resolution or mutation.
- Plugin reinstall now reports a localized not-installed diagnostic before reading local plugin metadata instead of exposing a raw missing `plugin.json` path error.
- Product-command tables now render explicit JSON `null` response values as empty cells instead of `<nil>`.

## 0.3.1 - 2026-07-11

### Fixed

- Plugin metadata validation now rejects accepted CTyun `900` status guards based on error-envelope fields such as `error` or `errorCode`, so API failures surface as API status errors instead of later table-rendering errors.

## 0.3.0 - 2026-07-11

### Added

- Plugin metadata can now record deprecation notices for operations, commands, command options, and table fields while keeping documented surfaces available.
- Help and runtime output now warn when deprecated plugin commands, APIs, options, or displayed fields are used, while showing replacement guidance only for CLI-facing command or option replacements; runtime warnings can be disabled with `CTYUN_WARN_DEPRECATED=0` or `ctyun config set warn_deprecated false`.
- Plugin metadata can record an explicit API URI scope for its upstream ownership boundary.

### Changed

- Development builds now prefer bundled source-tree plugins over installed plugins with the same name when executing product commands.
- Live plugin commands that map request fields from the selected profile region now fail locally when neither profile `region` nor the exposed command input supplies the value.
- Region-style plugin commands with a trailing `{region_id}` argument can now omit that argument when the selected profile supplies `region`, without also exposing a duplicate `--region` option.

## 0.2.0 - 2026-07-05

### Added

- Plugin metadata can declare vertical table layouts, default column subsets, conditional option requirements, and guarded CTyun `900` accepted responses.

### Changed

- Single-object plugin tables can render as localized field/value rows, with default column subsets used when metadata declares them.
- Table selector help and diagnostics now distinguish horizontal columns from vertical fields, list selectors one per line, show declared defaults as separate markers, and keep stable keys available for `--cols`, `--filter`, and `--sort`.
- Object-valued table cells now use a consistent nested `key=value` format.
- Plugin discovery can list or search all registry channels with `--channel all`, while install and update commands validate that a concrete channel was selected.
- Plugin command parameter errors now omit internal command IDs and focus on the actionable flag or condition.
- Plugin help annotates conditionally required options while keeping them optional in usage until their condition applies.
- Top-level help now acknowledges the official CTyun `ctyun-cli` released on 2026-07-02 and positions this project as an unofficial alternative.
- Development builds still discover bundled source-tree plugins, while release builds now load only installed plugin bundles.

### Fixed

- Release packaging now builds with `-trimpath` so binary artifacts do not embed local checkout paths.

## 0.1.0 - 2026-07-04

### Added

- Global `--version` and `-v` flags alongside the existing `ctyun version` command.
- Plugin storefront commands now support installing, reinstalling, updating, listing, and searching hosted or development-bundled plugins.

### Changed

- Refined core and plugin help output so usage lines, argument sections, required options, option value placeholders, allowed values, defaults, and plugin discovery sections use one consistent format.

### Fixed

- Fixed core update and release comparisons to use full SemVer prerelease precedence, so stable `0.1.0` is treated as newer than `0.1.0-alpha.1`.
- Fixed plugin compatibility checks to use the same SemVer prerelease precedence, so a `0.1.0-alpha.1` core no longer accepts plugin constraints such as `>=0.1.0 <1.0.0`.
- Allowed `0.1.0-dev` source builds to satisfy the matching stable base compatibility range while keeping prerelease ordering intact.
- Fixed installer and update channel selection to prefer stable releases before beta and alpha when no channel is pinned.
- Improved table rendering for array and object values so multi-value fields are readable instead of Go's default space-separated formatting.
- Improved localized diagnostics for plugin metadata, registry, distribution, and API error paths.

## 0.1.0-alpha.1 - 2026-06-21

### Added

- Native `ctyun` CLI with localized help, table output by default, and JSON fallback.
- Config and profile commands with environment-variable credential precedence.
- Metadata-driven plugin command loading.
- Offline fixture mode for development and safe command-shape verification.
- Signed core self-upgrade metadata with GitHub and Gitee mirror selection.
- Signed hosted plugin registry metadata with GitHub and Gitee mirror support.
