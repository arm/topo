# Schema Compliance

The [machine-readable schema](schema/topo-project-specification.json) can be used to check a [Topo Project](../introduction/glossary.md#topo-project) against the Compose specification and the [`x-topo`](../introduction/glossary.md#x-topo) metadata requirements. It uses [JSON Schema](https://json-schema.org/) and is compatible with any supported tooling.

## Validating Schema Compliance in Your Editor

The schema can be associated with a Project Compose file using a [yaml-language-server](https://github.com/redhat-developer/yaml-language-server) modeline comment. Add the following line to the top of your `compose.yaml`:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/arm/topo/refs/heads/main/docs/project-specification/schema/topo-project-specification.json
services:
  # ...

x-topo:
  # ...
```

This works in any editor that uses yaml-language-server, including:

- **VS Code** — install the [YAML extension](https://github.com/redhat-developer/vscode-yaml) and check the Problems panel for schema errors.
- **Vim / Neovim** — configure [yaml-language-server](https://github.com/redhat-developer/yaml-language-server) as an LSP source using your preferred LSP client.

## Validating Schema Compliance using CLI

You can validate any Project `compose.yaml` from the command line with `check-jsonschema` in your own virtual environment:

```sh
pip install check-jsonschema
check-jsonschema --schemafile schema/topo-project-specification.json path/to/compose.yaml
```
