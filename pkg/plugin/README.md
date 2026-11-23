# Configuration actions

Interactive configuration discovers installed `topo-plugin-*` executables in the
`topo/plugins` subdirectory of the OS user configuration directory. Use
`topo configure --plugin-dir /path/to/plugins` to override it (`clone` also supports
this flag). Non-interactive configuration does not load plugins.

Suggestions providers receive project metadata in `Request.Project` (currently
`Name`, taken from `x-topo.name`), the parameter's advisory `hints`, current value,
and plugin observations when a target is supplied. The same project metadata is
included in `RunAction` requests. If `x-topo.name` is absent, the name is empty.
Available actions appear in the prompt footer, starting with `Ctrl+O`. Actions
run only when explicitly selected and receive a fresh snapshot of the edited value.

Actions run in the background: typing, Enter, and Ctrl+C remain available. The
first returned suggestion fills the input for editing and confirmation. Errors
are displayed without ending the prompt. Accepting a value or cancelling the
prompt cancels its outstanding actions and waits for cleanup.

`RunAction` implementations must observe their context and release resources
before returning. A plugin that does not finish cancellation within two seconds
loses its RPC connection.

## Transport compatibility

The experimental transport uses `BeginAction` and `CancelAction` RPCs and a
per-invocation ID on `RunAction`. Registration happens before execution so
cancellation cannot race ahead of registration. Completion of `RunAction`
acknowledges cleanup. Rebuild plugin executables against the matching SDK when
testing changes to the experimental transport.

## AI Portal example

Run `bash pkg/plugin/examples/build-plugins.sh` to build the AI Portal and memory
examples, then pass `--plugin-dir pkg/plugin/examples/bin` to `topo configure`.
Set `TOPO_AI_PORTAL_URL` if the Portal that supports the handoff is not running at
`http://localhost:3000/ai/models`.

The example offers model selection for a parameter with a nonempty `model.task`
or `model.format` string, or `huggingface.repo-id` in its `meta.type` hint (a
string or array). It opens the browser only when selected and returns a repository
ID as a candidate. Closing the browser alone does not end the action; you can
still enter a value manually. The plugin closes its loopback callback server on
cancellation.
