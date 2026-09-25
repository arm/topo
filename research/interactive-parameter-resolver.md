# Interactive parameter resolver exploration

## Question and status

Can Topo provide useful inline parameter forms without adopting the Charm dependency stack? The alternative must preserve the existing visual language while remaining small enough for the team to maintain. Dependency ownership matters more than binary size alone.

Two [throwaway prototypes](../internal/parameter/prototype/README.md) now support sequential prompts and an inline navigable form. No implementation has been selected. Production code and its dependency files are unchanged.

## Agreed behavior

- Enter accepts typed text. With a blank input, Enter keeps the current value without writing it.
- A blank optional field without a current value stays missing. A blank required field without a current value cannot advance.
- Required means present, not non-empty. An explicit empty update and an empty current value both satisfy requiredness.
- A separate action sets an empty string. The prototype uses Ctrl+X and advances immediately.
- Current values appear as ghost text. Right arrow on a blank input copies the current value into the editor. Deleting all typed text restores ghost text.
- Display the winning source only: an environment file path or the process environment. Keeping an environment value means no write.
- No special handling for secrets is included in this exploration.
- No full-screen interface. Preserve Topo colors and its `NO_COLOR` policy.
- `golang.org/x/term` is acceptable for the non-Charm implementation.

The prototype-specific choices are Ctrl+X, Right arrow, Tab behaving like Enter, cancellation discarding all draft updates, and preservation of leading and trailing spaces. These can change after interaction review.

## Current production behavior

[`InteractiveResolver`](../internal/parameter/interactive_resolver.go) uses a line scanner, trims whitespace, and ignores empty input. The result map already represents no update through an absent key and explicit empty through a present key with `""`.

[`StrictResolverChain`](../internal/parameter/strict_resolver_chain.go) currently treats empty as missing. A production implementation must change shared validation, not just the interactive prompt. CLI input such as `NAME=` must follow the same presence semantics.

[`env.CurrentValues`](../internal/env/env.go) merges environment files and overlays process environment values, discarding origin information. Provenance must be captured during loading. It cannot be inferred from the final string, especially when multiple sources contain the same value.

Compose's [environment loader][compose-loader] evaluates files in sequence against accumulated values. Recording origins must preserve that interpolation behavior. Initially, source means the file containing the winning assignment, not every variable that contributed to an interpolated value.

The existing [`term` package](../internal/output/term/palette.go) supplies colors, headers, and terminal detection. It does not supply an editor, key decoder, or redraw engine.

## Approaches considered

| Approach | Benefit | Cost or limitation |
| --- | --- | --- |
| Enhance the existing line scanner | Minimal implementation and dependencies | Colors and retry validation are easy. Editable ghost text and immediate action keys need a different input mechanism. Text commands introduce escaping or reserved-input rules. |
| Use `huh` | Existing input, select, validation, focus, and viewport behavior | Needs explicit keep versus empty state, a theme, and acceptance of its transitive dependencies |
| Custom form on Bubble Tea and Bubbles | Own form semantics without owning terminal event handling | Still adopts the Charm stack. Bubbles text input imports Lip Gloss. |
| Custom form on `x/term` | Small dependency footprint and direct use of Topo styling | Own editing, escape decoding, rendering, and terminal compatibility |
| Custom OS terminal support | Full ownership | Adds platform-specific work that `x/term` already handles. Not pursued. |

[`x/term.Terminal`][term] offers a line editor and completion callback, but not a ready-made placeholder or form interface. The native experiment instead uses raw-mode primitives to make the implementation cost visible.

[Promptkit][promptkit] supplies customizable prompts but also builds on Charm and warns that its interface is unstable. [Survey][survey] declares itself unmaintained. Neither offers a compelling alternative for the dependency-ownership question.

## Findings from Huh source and implementation

The prototype pins `huh` v2.0.3, Bubble Tea v2.0.2, and Bubbles v2.0.0.

- Huh can run on the normal screen. Its [default form setup][huh-form] does not enable the alternate screen. `Input.Inline(true)` means placing the title and input on the same line.
- A [placeholder][huh-input] is visual, not a value or a keep action. Stock input returns a string and validates it on advancement.
- The prototype wraps `huh.Input` to track explicit empty intent, fill the editor from the current value, reject multiline paste, and provide action help. It does not fork Huh.
- [Themes][huh-theme] use Lip Gloss. Matching Topo ANSI colors is practical without removing Lip Gloss or using Charm's default theme.
- The [accessible string prompt][huh-accessible] trims input and substitutes a default for blank input. The v2.0.3 form's accessible runner also discards field errors. The PoC rejects unsupported terminal operation rather than silently switching to this behavior.
- [Bubbles input][bubbles-input] supplies editing, paste handling, horizontal scrolling, and Unicode-related rendering machinery that the native implementation must otherwise own or obtain separately.

## Measurements

Measured on macOS Arm64 with Go 1.26.6. Module and package counts come from each command's imported dependency graph, excluding the standard library and Topo itself. They are not counts of every module declared in the shared prototype `go.mod`. Other target platforms can import different packages.

| Measurement | Native | Huh |
| --- | ---: | ---: |
| Imported external modules | 2 | 26 |
| Imported external packages | 2 | 40 |
| Implementation Go lines, including comments and blanks | 490 | 203 |
| Standalone stripped executable, bytes | 2,080,530 | 4,648,434 |

Both share another 169 Go lines for fixtures, decision semantics, command setup, and JSON reporting. Line counts describe these particular prototypes, not equivalent production completeness. The Huh count includes `x/term`, used by the shared terminal checks.

Native imports only `golang.org/x/term` and `golang.org/x/sys` beyond Topo. Windows console output setup also uses `x/sys`. The larger Huh graph includes its runtime, widgets, styling, clipboard access, Unicode width and segmentation packages, theme data, and terminal compatibility packages.

To inspect imported modules and reproduce a stripped build from the prototype directory:

```sh
go list -deps -f '{{if and .Module (not .Module.Main)}}{{.Module.Path}}{{end}}' ./cmd/native | sort -u
go build -trimpath -ldflags='-s -w' -o /tmp/topo-parameter-native ./cmd/native
```

Replace `native` with `huh` for the comparison. These are standalone prototype sizes, not measured increases to the production Topo binary.

## Verification and remaining implementation costs

Both variants passed macOS pseudo-terminal checks for all agreed value-state transitions, backward navigation, spaces, Latin UTF-8 paste, multiline-paste rejection, `NO_COLOR`, Ctrl+C cancellation, and SIGTERM cleanup. Checks confirmed no alternate-screen entry and restoration of terminal input modes. Resize and long-input checks established liveness and correct final decisions, not correct visual reflow.

Both variants were built for macOS, Linux, and Windows on Arm64 and x86-64. Linux and Windows interactive behavior remains unverified. Existing production tests for `parameter`, `env`, `project`, and `output/term` pass. The prototype module passes `go vet ./...`.

Native deliberately does not solve grapheme editing or Unicode display widths. It counts code points, so wide characters, combining marks, and emoji can misalign rendering. It also has a bounded escape decoder, clipped help at narrow widths, no suspend/resume support, and no plain-text accessibility fallback. These are real ownership costs, not equivalent functionality obtained for fewer lines.

A production native implementation would need a decision on Unicode support: accept a small maintained width/segmentation dependency, maintain that logic ourselves, or explicitly restrict supported input. A successful ASCII demo does not settle that choice.

Closed-choice controls were not implemented in this first experiment. Huh already supplies selection controls. The native version would need selection state, navigation, and rendering. Add one concrete choice field after evaluating the basic interaction, rather than designing a schema-driven form framework now.

## Semantics to preserve during production integration

Setting empty does not remove an assignment. It also does not guarantee an empty application value: Compose `${NAME:-fallback}` treats empty like missing, while `${NAME-fallback}` preserves empty. See [Compose interpolation][compose-interpolation].

A process environment variable can continue to override a newly saved file value. The agreed scope displays only the winning source, not a shadowing warning or write destination. Do not silently persist current environment values when the user chooses keep.

Keep requiredness and later type or closed-set validation in Topo logic shared by CLI and interactive resolvers. Presentation libraries should not define those rules.

## Evaluation still needed

- Which layout feels better: a persistent question transcript or backward navigation within an inline form?
- Are Right arrow and Ctrl+X discoverable enough? Does immediate advancement after setting empty feel correct?
- Does seeing ghost text again after deleting a draft clearly communicate keep rather than empty?
- Is the native editing experience sufficient once its limitations are visible?
- Which terminal and accessibility guarantees are mandatory before production adoption?
- Does a small Unicode dependency offer a better ownership balance than either a limited editor or the complete Charm stack?

Verdict: pending hands-on review. The native implementation is small enough to evaluate seriously, but the current size comparison is not evidence of production parity.

[compose-loader]: https://github.com/compose-spec/compose-go/blob/v2.15.0/dotenv/env.go
[compose-interpolation]: https://docs.docker.com/reference/compose-file/interpolation/
[term]: https://pkg.go.dev/golang.org/x/term
[huh-form]: https://github.com/charmbracelet/huh/blob/v2.0.3/form.go
[huh-input]: https://github.com/charmbracelet/huh/blob/v2.0.3/field_input.go
[huh-theme]: https://github.com/charmbracelet/huh/blob/v2.0.3/theme.go
[huh-accessible]: https://github.com/charmbracelet/huh/blob/v2.0.3/internal/accessibility/accessibility.go
[bubbles-input]: https://github.com/charmbracelet/bubbles/blob/v2.0.0/textinput/textinput.go
[promptkit]: https://github.com/erikgeiser/promptkit
[survey]: https://github.com/AlecAivazis/survey
