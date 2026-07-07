# Core Migration Engineer Report

## Implemented

- Ordered task dispatch from parsed config arrays.
- `defaults` replacement semantics.
- Aggregate success tracking with `--exit-on-failure`.
- Built-in directive handlers for `create`, `link`, `shell`, and `clean`.
- Link options for `path`, `type`, `create`, `relink`, `force`, `backup`, `relative`, `canonicalize`, `if`, `ignore-missing`, `glob`, `exclude`, and `prefix`.

## Behavior Preservation

- Existing regular files block symlink creation unless `force` or `backup` semantics apply.
- Missing targets fail unless `ignore-missing` is set.
- Dry-run emits actions without performing filesystem mutations.
- Multiple config files are concatenated before dispatch.

## Documented Drift

- Dynamic Python plugins are omitted from the Go port.
- Recursive glob support is implemented locally for common `**` Dotbot patterns; it is intentionally minimal compared with Python's `glob.glob`.
