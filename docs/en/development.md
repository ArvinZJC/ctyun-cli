# Developing and testing ctyun

[Project home](../../README-EN.md) · [简体中文](../zh-CN/development.md)

## Follow a request through the core

Start at the repository root. This example runs a complete product command against the captured Region response. It needs Go and the project dependencies. The output should contain the row for 保定, with region code `cn-baoding-1`.

```sh
go run ./cmd/ctyun region list --name 保定 --cols region_name,region_code --offline
```

| If your change affects… | Start here | What to verify |
|---|---|---|
| Command dispatch, options or help | [CLI entry point](../../internal/cli/cli.go) and [help](../../internal/cli/help.go) | Group help, leaf argument errors and both empty-token and partial-option completion |
| Configuration or language selection | [Configuration resolution](../../internal/config/resolution.go) | Effective value, winning source and credential masking in diagnostics |
| Request construction or authentication | [Command bindings](../../internal/cli/command_bindings.go), [client](../../internal/client) and [signing](../../internal/signing) | Query, body, headers and path encoding against the operation contract |
| Output | [Command tables](../../internal/cli/command_tables.go) and [renderer](../../internal/output) | Localised labels, stable keys and terminal width with interactive and redirected output |
| Plugin discovery or installation | [Registry](../../internal/registry), [distribution](../../internal/distribution) and [local installation](../../internal/plugin/install.go) | Signed entry, checksum, manifest identity, compatibility and safe extraction |
| A product command or waiter | [Plugin authoring](plugin-authoring.md) | Source evidence, promoted metadata and real-bundle fixture checks |

For a core change, exercise the public command via `cli.Execute` or `cli.Run` using synthetic metadata. For a product change, put the regression case in [tools/plugincheck](../../tools/plugincheck) and exercise the tracked bundle. In particular, a successful fixture run verifies the command and its recorded response shape. Only a live retrieval can verify the endpoint, account permissions and current service response.

## Preparing a contribution

Use the same option declarations for parsing, help and completion. Put help text in the help catalogue, runtime diagnostics in the message catalogue, and product labels in the plugin locales or table definition. Preserve machine values such as request field names in each language. Document public changes in both language editions. Use the [project rules](../../AGENTS.md) for code comments, file sizes and commit subjects, and the [release guide](releases.md) to decide which changelog to update.

## Go toolchain

The minimum Go version for development and builds is 1.26.0.

The project follows the [upstream Go support window](https://go.dev/doc/devel/release#policy) by default, supporting the two latest Go release families and normally using the older family's `.0` release as the minimum. Language features, standard-library APIs, dependencies, or necessary compiler/runtime fixes may require a higher minimum; outdated dependencies should not be retained indefinitely to support end-of-life Go versions. The `go` directive in `go.mod` records the minimum requirement; development and release builds use the latest patch of the latest stable Go release. A local toolchain upgrade alone does not require raising the minimum or adding a `toolchain` directive.

Use Go’s shared build and module caches for routine development. The coverage tool honours the same cache settings as other Go commands. For multi-platform releases, repeated instrumented builds, or a read-only shared build cache, use a temporary build cache for the batch. The following example removes it when the subshell exits:

```sh
(
  build_cache=$(mktemp -d) || exit 1
  trap 'rm -rf "$build_cache"' EXIT
  export GOCACHE="$build_cache"
  go run ./tools/coverage
)
```

Positional argument placeholders use braces, such as `{instance_id}`; option values and developer shorthand such as `<name>` and `<plugin-command>` use angle brackets. Replace placeholders with actual values, commands, or paths before running the examples.

## Build and test

To build and run a standalone development binary without leaving an executable in the checkout, run this from the repository root.

```sh
(
  build_dir=$(mktemp -d) || exit 1
  trap 'rm -rf "$build_dir"' EXIT
  go build -o "$build_dir/ctyun" ./cmd/ctyun && "$build_dir/ctyun" --version
)
```

```sh
go run ./cmd/ctyun region list --offline
go run ./cmd/ctyun region list --fixture
```

`--offline` and `--fixture` both enable bundled plugin fixtures and do not call live CTyun APIs. They are long-only product-command options for development builds, must follow the complete product command path, and are not global options. Release builds neither recognise nor expose these development options.

To inspect a live request, run `go run ./cmd/ctyun --debug region list` with credentials authorised for that query. HTTP diagnostics go to stderr. Fixture runs read a local response and generate no HTTP trace.

Development builds can use `--bundled` to search, list, install, reinstall, or update plugins from in-tree plugin metadata. The default channel is `stable`. The ECS examples below select `beta`, the channel declared in its manifest. Product command execution in development builds also prefers in-tree bundled plugins, so local metadata changes remain visible even when a released plugin with the same name is installed. Like `--fixture`, `--bundled` is for development and test workflows and is omitted from regular help.

```sh
go run ./cmd/ctyun plugin list --available --channel all --bundled
go run ./cmd/ctyun plugin search ecs --channel beta --bundled
go run ./cmd/ctyun plugin install ecs --channel beta --bundled
go run ./cmd/ctyun plugin reinstall ecs --channel beta --bundled
go run ./cmd/ctyun plugin update ecs --channel beta --bundled
```

Run the test suite as an unprivileged user. Several tests rely on the operating system denying writes to read-only files or directories.

```sh
git ls-files -co --exclude-standard '*.go' | sort -u | xargs gofmt -w
go vet ./...
go test ./internal/cli -run '^TestGoFilesStayUnderLineLimit$'
go test -timeout=30m ./...
go test ./internal/cli -run Completion -v
go test -timeout=30m ./tools/plugincheck
go run ./tools/coverage
```

The coverage gate requires 100% statement coverage of the unfiltered profile at `.cache/coverage/raw.out` for the current Go platform. It checks execution counts directly, with no file or line exclusions and no rounding allowance. Tests include the command entry points, maintenance tools, archive helpers, and filesystem failure paths.

After plugin changes, verify according to the affected area. Lint the changed plugin first, then run the matching offline command. For changes to shared plugin loading, command parsing, or table rendering, also run the corresponding Go package tests.

```sh
go run ./cmd/ctyun plugin lint ./plugins/<name>
go run ./cmd/ctyun <plugin-command> --offline

go test -timeout=30m ./tools/plugincheck
go test ./internal/cli ./internal/plugin ./internal/output
```
