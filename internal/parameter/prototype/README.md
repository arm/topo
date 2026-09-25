# Interactive parameter resolver prototypes

Throwaway experiments, not production resolver implementations. The question is whether a small prompt built with `golang.org/x/term` and Topo styling is practical to maintain, compared with a customized `huh` form. Both implementations expose the same keep, set, and explicit-empty semantics in sequential and inline-form layouts.

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

Both layouts use the normal terminal screen, not the alternate screen. Sequential mode leaves completed decisions in scrollback. Form mode redraws its own region and supports backward navigation.

## Use the controls

| Key | Action |
| --- | --- |
| Enter or Tab | Accept typed text, or keep the current value when the input is blank |
| Ctrl+X | Set an explicit empty string and advance |
| Right arrow with blank input | Copy the current value into the input for editing |
| Shift+Tab | Return to the previous field in form mode |
| Ctrl+U | Clear the entire draft and restore the current-value ghost text |
| Left, Right, Home, End | Move within the draft |
| Backspace, Delete | Delete text |
| Ctrl+C or Ctrl+D | Cancel without returning updates |

A blank required field with no current value cannot advance. Ctrl+X satisfies requiredness because an empty string is present. Keeping a current environment value never creates an update.

Deleting all typed text restores ghost text. A current empty string appears as `<empty current value>`. Revisiting an explicitly empty update shows `<set empty>`. These are display labels, not magic input strings. Typing either label stores it literally.

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
3. Press Ctrl+X to set `LABEL` to empty.
4. Press Enter on `NOTE` and `EMPTY`.

The final report must contain exactly this update map:

```json
"updates": {
  "LABEL": ""
}
```

On a second run, try Right arrow on `GREETING`, edit the value, clear it with Ctrl+U, and press Enter. The original value must remain unchanged, without an update. In form mode, use Shift+Tab to revisit an earlier answer. Try cancellation after accepting several fields.

For an uncolored comparison, set `NO_COLOR=1`. Input and stderr must be terminals. `TERM=dumb` is rejected because neither prototype implements a plain-text fallback.

## Understand the limitations

- Native rendering treats each Unicode code point as one terminal column. Accented Latin text was exercised, but wide characters, combining marks, and emoji can misalign the cursor or clipping. Deletion is by code point, not grapheme cluster.
- Native mode needs at least 30 columns and 10 rows. It clips help and metadata at narrow widths. It polls dimensions and supports basic horizontal scrolling, but terminal reflow during resize is not visually verified.
- Native key decoding covers a small set of conventional escape sequences. It is not a general terminal input library. History, word editing, mouse input, completion, and suspend/resume are not implemented.
- Only bracketed paste can be distinguished from typing. An unbracketed pasted newline can act like Enter.
- Native input is limited to 4096 code points, with a 1 MiB pending-paste bound. Huh uses its text input character limit and terminal input machinery.
- The Huh prototype adds a wrapper around `huh.Input` to retain update intent. Its accessible fallback is deliberately disabled because that path does not preserve the agreed empty-input semantics.
- Windows output-mode setup is included. Linux and Windows were cross-compiled, not exercised interactively.
- Neither prototype changes production requiredness, implements provenance loading, or adds schema types and closed-choice controls.

## Verification and findings

Both implementations passed macOS pseudo-terminal exercises for required-as-present validation, explicit empty updates, no-write keeps, editing ghost values, back navigation, preserved spaces, UTF-8 paste, rejected multiline paste, cancellation, terminal-mode restoration, and `NO_COLOR`. Long-input and resize checks verified liveness and decision state, not visual correctness. No alternate-screen entry was emitted.

Both build for macOS, Linux, and Windows on Arm64 and x86-64. `go vet ./...` passes within this module. Existing production tests for `parameter`, `env`, `project`, and `output/term` still pass.

See [research notes](../../../research/interactive-parameter-resolver.md) for measurements, source references, and outstanding decisions. After evaluating the interactions, keep the findings and delete these prototypes or replace them with a production implementation.
