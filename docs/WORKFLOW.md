# Workflow Guide

This guide covers how `schemagen` fits into normal developer workflow.

## Typical Loop

1. update database schema
2. adjust `schemagen.yaml` if needed
3. adjust `schemagen.relations/` if relation mapping changed
4. regenerate entities
5. review the diff

Example:

```bash
schemagen generate --config schemagen.yaml
```

## Safe Regeneration Model

Generated files use managed `SECTION` markers.

That means:

- generated sections are replaced on regen
- manual code outside those markers is preserved

This is the core safety model of the tool.

## Conflict Handling

When a file already exists and does not look like a managed schemagen file, behavior depends on `on_conflict`.

- `skip` -> keep the file, log a warning
- `error` -> fail immediately
- `backup` -> move old file to `*.bak.<timestamp>`
- `overwrite` -> replace the file directly

Recommended team default:

```yaml
on_conflict: skip
```

Recommended local iteration default:

```yaml
on_conflict: backup
```

## Logging Modes

### Default

Good for everyday use.

Shows:

- final summary
- warnings

### `--verbose`

Use when:

- debugging generation flow
- checking which tables are processed
- investigating skipped files

### `--quiet`

Use when:

- scripting
- CI wrappers
- you only care about failure output

## Recommended Review Habit

Do not treat generated code as unreviewable.

Always review:

- renderer choice
- nullable behavior
- JSON casing
- decimal and JSON strategy
- relation field names
- `type_overrides`

## Suggested Per-Renderer Workflow

### `sqlx`

Recommended:

- enable `generate_field_refs`
- use explicit `type_overrides` for strong domain columns
- keep relation config explicit

### `gorm`

Recommended:

- use `nullable_strategy: pointer`
- review relation names carefully
- rely on built-in `gorm.DeletedAt` handling for `deleted_at`

### `plain`

Recommended:

- keep relation config lean
- use it when you want clean generated structs without DB-library coupling

## Upgrading Existing Projects

Current relations loading is backward-compatible:

- preferred: `schemagen.relations/`
- fallback: `schemagen.relations.yaml`

Migration path:

1. create `schemagen.relations/`
2. split the legacy file by domain
3. keep one table owned by one file
4. once verified, remove the legacy file

## When to Add Type Overrides

Use overrides when the default mapper is compile-safe but domain-weak.

Examples:

- UUID should be `uuid.UUID`
- money should be `decimal.Decimal`
- MariaDB JSON needs explicit `json.RawMessage`
- a specific column should map to a domain type

Do not add overrides everywhere just because you can. Keep them targeted.
