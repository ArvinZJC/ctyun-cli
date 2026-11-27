# Running product commands

[Project home](../../README-EN.md) · [简体中文](../zh-CN/commands.md)

These commands do not depend on product plugins. They are useful right after installation for checking the version, reading help, generating completion scripts, or checking network connectivity:

```sh
ctyun --version
ctyun help
ctyun help config
ctyun completion zsh
ctyun doctor local
ctyun doctor network
```

Plugin command help becomes available after installing the matching plugin, for example `ctyun help region list`.

## Read the help for the installed version

Run `ctyun region` to see its subcommands, then `ctyun region list --help` to see the options and columns for that query. In usage lines, replace `{name}` with a positional argument and `<value>` with an option value. Square brackets mark optional inputs. For a command that requires a resource pool, select its ID from `ctyun region list` and save it in your profile or pass `--region` if the command offers it.

To enable completion for the current Bash session, run `source <(ctyun completion bash)`. Fish users can run `ctyun completion fish | source`. For Zsh, save `ctyun completion zsh` as `_ctyun` in a directory on `fpath` before calling `compinit`. PowerShell users can load the generated script with `ctyun completion powershell | Out-String | Invoke-Expression`. Product suggestions reflect the plugins available to the CLI.

## Product commands

Dangerous operations prompt for `y/N` confirmation by default; scripts can use `--yes` or `-y` to skip the prompt.

Install the Region and ECS plugins before running these examples. For ECS, set the resource pool in your profile or supply `--region <id>` on each command. Replace the example instance ID with one returned for your account.

```sh
ctyun region list
ctyun --lang en-GB region list --name 华东1 --cols "Region ID,Region Name,Region Code"
ctyun --lang en-GB ecs instance list --cols "Instance ID,Name,Status"
ctyun ecs instance list --name api-test01
ctyun ecs instance show c5a7966a-88e7-362b-6e11-c2d8fbfc07ca
```

## Output and filtering

- `--cols`, `--filter`, and `--sort` accept the column labels shown in the table, while stable column keys remain supported. When labels overlap, use the qualified label shown in help or the stable key.
- Quote column labels that contain spaces and filter expressions that contain shell metacharacters. For example, `--filter 'Name=*'` matches a displayed name whose literal value is `*`.

```sh
ctyun ecs instance list --output json
ctyun ecs instance list --table compact
ctyun ecs instance list --table plain
ctyun ecs instance list --no-header
ctyun --lang en-GB ecs instance list --filter Status=running --sort "-Instance ID"
```

Interactive tables measure Chinese, English, emoji, and other Unicode content by terminal display width and, where possible, wrap at whitespace or common machine-value separators; redirected or piped output retains its natural width. The `bordered`, `compact`, and `plain` styles share the same column-width calculation and wrapping rules.

## Waiting for resource or task state

Waiters require core `>=0.5.0 <1.0.0`. Use a retrieval command with `--wait <waiter>` to poll until its documented target state is observed. Command help lists applicable waiters, and shell completion offers the same choices. Waiters are checked before a request; incompatible commands and operations without safe polling metadata are rejected. Polling repeats the retrieval command, so use the resource or task identifier returned by the earlier mutation.

```sh
ctyun ecs instance show {instance_id} --wait ecs.instance.running
ctyun vbs backup show --backup-id <backup_id> --wait vbs.backup.available
ctyun hpfs dataflow-task show --task-id <task_id> --wait hpfs.dataflow-task.completed
ctyun evs snapshot list --snapshot-id <snapshot_id> --wait evs.snapshot.available
ctyun ims image show {image_id} --wait ims.image.active
```

For waits on a collection, provide the resource identifier required by command help. On each poll, the waiter checks the state of the one resource with that exact identifier. It keeps polling if no resource matches and reports an error if multiple resources match.

The first response counts as one poll. Each waiter sets a maximum number of polls and a delay between them. A single poll can make more than one HTTP request when a retrieval is retried. The HTTP timeout applies separately to each request. Both the polling interval and the request timeout support up to 9,223,372,036 whole seconds on 64-bit platforms. Null and unknown values stay pending until the limit. Failure and timeout are printed as final waiter states; they currently do not change the command exit status. JSON output remains the initial response on stdout, with the waiter status on stderr.

## Use the result in a script

A product option such as `--name` on `ctyun region list` sends a query parameter to the service. Table controls work on the rows returned by that request. In English, use `--filter 'Region Name=保定'` for an exact match or `--filter 'Region Name!=保定'` to exclude that value. Sorting compares displayed strings and a leading `-` reverses their order. For a script that runs in several interface languages, the same column has the stable key `region_name`.

JSON preserves the response fields. For example, with the Region plugin installed and credentials configured, this Bash pipeline prints an ID and a name per row. It also requires `jq`.

```bash
set -o pipefail
ctyun region list --output json | jq -r '.returnObj.regionList[] | [.regionID, .regionName] | @tsv'
```

For an automated wait, read the final waiter state before proceeding. A successful process exit alone does not establish that the resource reached the target state.
