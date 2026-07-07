# ADR-001: Support YAML, JSON, JSON5, and TOML Configs

## Status

Accepted

## Date

2026-06-21

## Context

`dotbot-go` executes directives in config source order. YAML, JSON, JSON5, and TOML now have ordered parsing paths. HOCON support depended on a parser API that exposed objects as Go maps, which made directive order nondeterministic unless a deeper parser implementation was added and maintained.

The project already supports four config formats that cover the common dotfiles use cases:

- YAML for classic Dotbot-style configs.
- JSON for strict machine-generated configs.
- JSON5 for JSON-like configs with comments and trailing commas.
- TOML for users who prefer TOML syntax.

## Decision

Remove HOCON config support and the `github.com/gurkankaymak/hocon` runtime dependency.

Supported config extensions are now:

- `.yaml`
- `.yml`
- `.json`
- `.json5`
- `.toml`

The `.conf` and `.hocon` extensions are unsupported. Existing `install.conf.yaml` filenames remain supported because the actual extension is `.yaml`.

## Alternatives Considered

### Keep HOCON With Sorted Fallback

- Pros: Avoids breaking users with `.conf` or `.hocon` configs.
- Cons: Violates the source-order execution guarantee and behaves differently from the other supported formats.

Rejected because deterministic directive order is more important than keeping a format with weaker semantics.

### Build A Custom Ordered HOCON Parser

- Pros: Full format parity with deterministic order.
- Cons: Higher implementation and maintenance cost for a less common dotfiles format.

Rejected because YAML, JSON, JSON5, and TOML already cover the target use cases.

## Consequences

- The dependency graph is smaller.
- The config parser contract is simpler: all supported formats preserve directive order.
- Users with HOCON configs must migrate to YAML, JSON5, or TOML.
- `.conf` files now fail with `unsupported config file format`.
