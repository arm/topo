# Breaking change policy

This policy defines which changes to the Topo are breaking changes. It helps reviewers classify changes consistently before a release.

This policy explicitly does not extend to:

- Hidden commands and flags
- Experimental features
- Human-readable output, help text, warnings, and logs
- Source code APIs and packages
- Build tools and developer scripts
- Undocumented or malformed input that the binary happened to accept

A hidden or experimental interface becomes part of the compatibility contract when it is promoted to a documented, supported interface.

## CLI compatibility

The following CLI changes are breaking:

- Removing or renaming of a supported command, flag, positional argument, or environment variable consitutes a breaking change.
- Making an optional flag or environment variable required

The following CLI changes are not breaking:

- Adding a command, flag, positional argument, or environment variable without changing existing behavior
- Renaming an interface while retaining the old form as a functional alias
- Changing help text or completion descriptions

## JSON compatibility

JSON structure is contractual. JSON values are not contractual unless their allowed values are explicitly documented as a closed set.

The following JSON changes are breaking:

- Removing or renaming a field
- Making a previously required/non-nullable field optional/nullable
- Changing a field type or nesting
- Changing documented array ordering semantics
- Adding a value to an explicitly documented closed set of allowed values

The following JSON changes are not breaking:

- Adding an optional field
- Making a previously optional/nullable field required/non-nullable
- Reordering object fields
- Changing field values, descriptive labels, or messages
- Changing array contents
- Adding a value when the allowed values are not documented as a closed set

Consumers must ignore unknown optional fields and must not depend on specific values unless the documentation defines those values as stable.

A JSON value change can still accompany a breaking behavior change. Evaluate the underlying behavior separately under this policy.

## Host and target requirements

A new or stricter requirement is breaking when a previously supported host or target no longer works.
This rule includes:

- Requiring a new executable, daemon, runtime, kernel capability, hardware feature, or network service
- Increasing a minimum dependency version
- Supporting fewer implementations, operating systems, or architectures
- Requiring new permissions, configuration, credentials, or network access

Removing a host or target requirement is not breaking if existing user capabilities remain available.

## Project and configuration formats

The following input format changes are breaking:

- Rejecting input that was valid according to the released documentation
- Removing or renaming a supported field
- Making an optional field required
- Narrowing an accepted type, range, or set of values
- Giving existing input incompatible semantics

The following input format changes are not breaking:

- Adding an optional field or accepted value
- Accepting additional input forms
- Warning that a supported input form is deprecated while continuing to accept it
- Rejecting malformed or undocumented input that happened to be tolerated

## Compatibility across upgrades

A newer binary must continue to read and operate on files created or modified by the previous released version. A change is breaking if it requires manual migration, silently discards data, or cannot read those files.

Automatic, lossless migration is not breaking. Topo does not guarantee that an older binary can read files after a newer binary migrates them unless rollback support is explicitly documented.

A newer binary must also continue to manage existing deployments. A change is breaking if users can no longer inspect, update, redeploy, or stop a deployment without recreating or manually repairing it.

## Exit status compatibility

The following changes are breaking:

- Changing an existing successful scenario from exit status `0` to a nonzero status
- Changing an existing failure scenario from a nonzero status to `0`
- Changing a specific nonzero status when that status is documented

Changing an undocumented nonzero status is not breaking. Error wording, log levels, timestamps, and diagnostic detail are not contractual.

JSON-formatted logs follow the JSON compatibility rules in this policy.

## Human-readable output

Human-readable output is not contractual. The following output can change without creating a breaking change:

- Wording and labels
- Formatting and ordering
- Progress indicators
- Warnings and logs
- Help text

User scripts are encouraged to consume JSON output when Topo provides it. Text is contractual only if the documentation explicitly defines it as machine-readable or suitable for shell evaluation.

## Deprecation and removal

Deprecation does not make a later removal non-breaking. It only communicates a future breaking change and gives users time to migrate.

## Bug fixes

Correcting behavior that clearly contradicts the documentation is not breaking, even if users may rely on the incorrect behavior. Changing documented behavior is breaking, even when the change is described as a fix.

For undocumented behavior, try to apply the general rules outlined in this policy.
