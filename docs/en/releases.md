# Preparing and verifying a release

[Project home](../../README-EN.md) · [简体中文](../zh-CN/releases.md)

## Decide what to version

Compare the shipped core and each plugin against their own preceding version tags. For a plugin, include new files and changes to the manifest fingerprint. Check that each proposed version tag is unused. A change confined to plugin metadata, repository tools or documentation can ship independently of a new core version.

Record changes to the shipped executable in the [core changelog](../../CHANGELOG.md). Each plugin has its own English changelog, such as the [Region changelog](../../plugins/region/CHANGELOG.md). Collect changes under `Unreleased` during development. When preparing a version, replace that heading with the version and date. Update the manifest, compatibility requirements, examples and both [plugin catalogues](catalogue.md) where affected.

## Validating the installers

From the repository root, run the fixture test on each target OS or on a matching CI runner (for example, `ubuntu-latest`, `macos-latest`, or `windows-latest` in GitHub Actions). The tests serve release archives over local HTTP and install into temporary directories.

```sh
go test -count=1 -v -run '^(TestBashInstallerSelection|TestWindowsInstallerLiteralPaths)$' ./tools/release
```

On macOS, Linux, or WSL, look for a pass from `TestBashInstallerSelection`. It exercises compact and indented indexes, channel selection and failure diagnostics. On Windows, `TestWindowsInstallerLiteralPaths` runs Windows PowerShell on paths containing spaces and brackets.

To test the checkout’s Bash script with a published Linux binary, run this from Bash or Zsh with a local Docker Engine configured for Linux containers. The Alpine-based Bash image exercises BusyBox `awk` without a separate JSON parser. The container runs as an unprivileged user and mounts the script read-only. Its default installation directory is removed with the container.

```sh
docker run --rm --user 65534:65534 -e HOME=/tmp -v "$PWD/install.sh:/work/install.sh:ro" \
  bash:5.2 bash -ec '
    bash /work/install.sh
    "$HOME/.local/bin/ctyun" --version
    "$HOME/.local/bin/ctyun" --help'
```

Compare the installed version with the installer’s success message. To test a specific mirror, add `-e CTYUN_INSTALL_SOURCE=github` or `-e CTYUN_INSTALL_SOURCE=gitee` before the image name. For a published-release check on macOS or Windows, use the appropriate command in the [installation guide](installation.md) with `CTYUN_INSTALL_DIR` set to a fresh temporary directory, then run that directory’s executable with `--version` and `--help`.

## Packaging and releases

The release packaging tool writes core binary archives, `core-index.json`, `core-index.sig`, installation scripts, plugin archives, `index.json`, and `index.sig`. Tests use local HTTP sources to verify signature and download behaviour. The published assets support installation, core updates, and plugin updates.

- The fixed `core` and `plugins` tags are the repository's only two GitHub Release pages and built-artefact roots: `core` stores core installation and update artefacts, while `plugins` stores plugin installation and update artefacts.
- Ordinary version tags continue to be created, but they do not receive separate GitHub Release pages or uploaded artefacts.
- The Gitee `core` Release keeps the same layout. Because Gitee limits attachments per Release, its fixed `plugins` Release stores only `index.json` and `index.sig`; each plugin archive is uploaded to an immutable Gitee Release for its existing `releases/plugins/<name>/<version>` tag. The signed Gitee index uses absolute URLs for those archives, so clients still download only the selected plugin.
- Actual versions and channels are selected by the signed `core-index.json` and `index.json`; version-specific change history is recorded in the canonical root and plugin changelogs.
- When reusing an output directory, the packager advances each core operating system and architecture pair independently within its channel. A partial release keeps the previous build available for each omitted platform until its replacement is published. The signed core index lists releases in descending SemVer order so installers select the newest available build for their platform. When two versions differ only in build metadata, the newly published build takes precedence for the platforms it supplies.
- Rebuilding an indexed core or plugin version refreshes that entry. A previously unindexed older core version can supply a platform that has no newer build. Packaging an older plugin version preserves the newer version advertised for that plugin and channel. Other channels keep their own entries.
- Each core version and each version of a given plugin belongs to one channel. The packager rejects a merged index that assigns the same version tag to multiple channels.
- Use `--out dist` for persistent local release builds. Reuse the current published indexes and plugin archives there so unchanged plugins retain their original compressed bytes and checksums across Go toolchain updates. The packager verifies the new core, GitHub plugin and Gitee plugin indexes, then removes superseded local archives whose names and SHA-256 checksums match the previous signed indexes. It keeps the archives needed for every indexed platform and channel. Finish uploading a release to both hosts before reusing its output directory for the next release.
- After updating fixed release assets, keep archives referenced by the current signed indexes, index signatures, and core installation scripts, and remove old archives that are no longer referenced; prerelease channel archives should stay when the index still advertises them.

The plugin index and archives at the output root are for GitHub. Use `gitee/index.json` and `gitee/index.sig` to replace the same-named files on Gitee's fixed `plugins` Release. `gitee/releases.json` is a publication manifest, not a user download: it records each archive's plugin, version, channel, target tag, checksum, and expected download URL. Verify existing Gitee version Releases and archives against the manifest and publish any missing ones before replacing the fixed `plugins` index.

Core and plugin versions must follow Semantic Versioning 2.0.0. Do not prefix release versions with `v`. Use versions like `0.1.0-alpha.1` with the `alpha`/`beta` channels for pre-releases, and versions like `0.1.0` with the `stable` channel for stable releases. The defaults in `internal/version/version.go` only identify unpackaged development builds, and release packaging overrides the actual version and channel.

Signed indexes also include a `versions` map derived from the final merged entries: `versions.<channel>` for core and `versions.<name>.<channel>` for plugins, each containing the highest semantic version for that channel. The README and plugin catalogue badges read these maps; the CLI continues to select downloads from the existing arrays.

Environment variables for packaging and distribution testing:

| Variable                    | Purpose                                                                                                                    |
|-----------------------------|----------------------------------------------------------------------------------------------------------------------------|
| `CTYUN_INSTALL_BASE_URL`    | Override the release root read by installation scripts for local or temporary release-asset validation                     |
| `CTYUN_RELEASE_PRIVATE_KEY` | Private key used by the release packaging tool to sign indexes                                                             |
| `CTYUN_RELEASE_PUBLIC_KEY`  | Overrides the embedded public key for core and plugin index verification whenever set, including in release builds |

For a local rehearsal, generate a temporary signing key and package the current platform into a fresh directory. The packager derives the public key from the private key and embeds it in the binary. For an existing distribution, use the private key whose public key is already trusted by installed clients.

```sh
go run ./tools/release --generate-key
release_dir=$(mktemp -d) &&
  CTYUN_RELEASE_PRIVATE_KEY="<private key from previous output>" \
  go run ./tools/release --version 0.5.2 --channel stable --out "$release_dir" --gitee-plugin-download-root "https://gitee.com/ArvinZJC/ctyun-cli/releases/download" --platform "$(go env GOOS)/$(go env GOARCH)"
```

For real releases, GitHub remains the canonical source and CI artefact authority, while Gitee is the synchronised mirror for more reliable access from mainland China. The installed `ctyun` verifies hosted core and plugin indexes using the signing public key and checks archive SHA-256 hashes; the bootstrap scripts use the [bootstrap trust model](installation.md).
