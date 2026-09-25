# Interactive parameter resolver prototypes

Throwaway experiments, not production resolver implementations. The question is whether a small prompt built with `golang.org/x/term` and Topo styling is practical to maintain, compared with a customized `huh` form. Both implementations expose the same keep and set semantics in sequential and inline-form layouts. Explicit-empty entry is deferred.

Nothing reads or writes project configuration. Fixtures, drafts, and decisions exist only in memory. Successful completion prints a JSON report to stdout. Prompts use stderr. Cancellation discards the report, including decisions made on earlier fields.

## Run a prototype

From the repository root:

```sh
go -C internal/parameter/prototype run ./cmd/native -layout sequential
go -C internal/parameter/prototype run ./cmd/native -layout form
go -C internal/parameter/prototype run ./cmd/huh -layout sequential
go -C internal/parameter/prototype run ./cmd/huh -layout form
```

The nested Go module isolates experimental dependencies from the production module. It references the repository only to reuse `internal/output/term`. Both commands share fixture and decision logic. The native command does not import Charm packages, even though the comparison module declares them.

Both layouts use the normal terminal screen, not the alternate screen. Sequential mode leaves completed decisions indented in scrollback, grouped together with one blank line before the active question. Form mode redraws its own region, indents inactive values beneath their field names, and supports backward navigation.

The heading reads “Configure project parameters.” Fields use a one-space left margin, with numbering beside the parameter name. Descriptions and examples use the normal foreground color directly above the input. The winning source appears beside the ghost value as a display-only annotation. It disappears when editing and is never copied into the value. Cyan highlights the active heading and input marker. Ghost text is subdued. There is no help bar. Standard editing and navigation shortcuts remain supported. Debug state is confined to the final report.

## Use the controls

| Key | Action |
| --- | --- |
| Enter or Tab | Accept typed text, or keep the current value when the input is blank |
| Right arrow with blank input | Copy the current value into the input for editing |
| Shift+Tab | Return to the previous field in form mode |
| Ctrl+U | Clear the entire draft and restore the current-value ghost text |
| Left, Right, Home, End | Move within the draft |
| Backspace, Delete | Delete text |
| Ctrl+C or Ctrl+D | Cancel without returning updates |

A blank required field with no current value cannot advance. An existing empty string satisfies requiredness because it is present. Keeping a current environment value never creates an update. The prototypes cannot set a value to an explicit empty string, and Ctrl+X has no action.

Deleting all typed text restores ghost text. A current empty string appears as `<empty string>`. This is a display label, not a magic input string. Typing the label stores it literally.

Leading and trailing spaces are preserved. These prototypes accept single-line text only. Bracketed paste containing control characters, including tabs and newlines, is rejected instead of silently transformed. No secret masking or special credential handling is implemented.

## Try the fixtures

| Parameter | Current value | Winning source | Required |
| --- | --- | --- | --- |
| `GREETING` | `Hello, World` | `.env.topo` | Yes |
| `REGION` | `west` | Environment | No |
| `LABEL` | Missing | None | Yes |
| `NOTE` | Missing | None | No |
| `EMPTY` | Empty string | `.env` | Yes |

The sources are fixture labels, not values loaded from your environment.

1. Keep `GREETING` and `REGION` by pressing Enter twice.
2. Press Enter on `LABEL`. The prompt must stay on that field.
3. Type `demo` and press Enter to set `LABEL`.
4. Press Enter on `NOTE` and `EMPTY`.

The final report must contain exactly this update map:

```json
"updates": {
  "LABEL": "demo"
}
```

On a second run, try Right arrow on `GREETING`, edit the value, clear it with Ctrl+U, and press Enter. The original value must remain unchanged, without an update. In form mode, use Shift+Tab to revisit an earlier answer. Try cancellation after accepting several fields.

For an uncolored comparison, set `NO_COLOR=1`. Input and stderr must be terminals. `TERM=dumb` is rejected because neither prototype implements a plain-text fallback.

## Understand the limitations

- Native rendering treats each Unicode code point as one terminal column. Accented Latin text was exercised, but wide characters, combining marks, and emoji can misalign the cursor or clipping. Deletion is by code point, not grapheme cluster.
- Native mode needs at least 30 columns and 10 rows. It clips metadata at narrow widths. It polls dimensions and supports basic horizontal scrolling, but terminal reflow during resize is not visually verified.
- Native key decoding covers a small set of conventional escape sequences. It is not a general terminal input library. History, word editing, mouse input, completion, and suspend/resume are not implemented.
- Only bracketed paste can be distinguished from typing. An unbracketed pasted newline can act like Enter.
- Native input is limited to 4096 code points, with a 1 MiB pending-paste bound. Huh uses its text input character limit and terminal input machinery.
- The Huh prototype wraps `huh.Input` for current-value editing and single-line paste validation. Its accessible fallback is deliberately disabled because it trims whitespace and drops field errors.
- Windows output-mode setup is included. Linux and Windows were cross-compiled, not exercised interactively.
- Neither prototype changes production requiredness, implements provenance loading, or adds schema types and closed-choice controls.

## Verification and findings

Both implementations passed macOS pseudo-terminal exercises for required-as-present validation, existing empty values, ignored Ctrl+X input, no-write keeps, editing ghost values, back navigation, preserved spaces, UTF-8 paste, rejected multiline paste, cancellation, terminal-mode restoration, and `NO_COLOR`. Long-input and resize checks verified liveness and decision state, not visual correctness. No alternate-screen entry was emitted.

Both build for macOS, Linux, and Windows on Arm64 and x86-64. `go vet ./...` passes within this module. Existing production tests for `parameter`, `env`, `project`, and `output/term` still pass.

See [research notes](../../../research/interactive-parameter-resolver.md) for measurements, source references, and outstanding decisions. After evaluating the interactions, keep the findings and delete these prototypes or replace them with a production implementation.
