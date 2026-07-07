# Config Formats Guide

`dotbot-go` supports YAML, JSON, JSON5, and TOML.

Most users should start with YAML because it is the closest to classic Dotbot examples.

## Supported Extensions

| Format | Extensions |
| --- | --- |
| YAML | `.yaml`, `.yml` |
| JSON | `.json` |
| JSON5 | `.json5` |
| TOML | `.toml` |

## Task Lists

A config file is an ordered list of tasks.

The order matters. This:

```yaml
- create:
    - "~/.config"

- link:
    ~/.config/nvim: nvim
```

means:

1. Create `~/.config`
2. Link `~/.config/nvim`

## YAML

YAML can use the task list directly:

```yaml
- defaults:
    link:
      create: true
      relink: true

- link:
    ~/.vimrc: vimrc
```

Full example: [examples/install.conf.yaml](../examples/install.conf.yaml)

## JSON

JSON also uses the task list directly:

```json
[
  {
    "defaults": {
      "link": {
        "create": true,
        "relink": true
      }
    }
  },
  {
    "link": {
      "~/.vimrc": "vimrc"
    }
  }
]
```

Full example: [examples/install.json](../examples/install.json)

## JSON5

JSON5 is like JSON, but comments and trailing commas are allowed:

```json5
[
  // Link Vim config.
  {
    link: {
      "~/.vimrc": "vimrc",
    },
  },
]
```

Full example: [examples/install.json5](../examples/install.json5)

## TOML

TOML is object-based. Put the ordered task list under `tasks`:

```toml
tasks = [
  { defaults = { link = { create = true, relink = true } } },
  { link = { "~/.vimrc" = "vimrc" } },
]
```

Keep inline tables on one line. This is a TOML syntax rule, not a `dotbot-go` rule.

Full example: [examples/install.toml](../examples/install.toml)

## Choosing A Format

Use YAML if you want the simplest path and compatibility with common Dotbot examples.

Use JSON if another tool generates the config.

Use JSON5 if you want JSON with comments.

Use TOML if your existing tooling prefers TOML.
