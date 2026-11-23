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
