# Configuration Guide

This guide explains the main knobs in `schemagen.yaml` and when to use them.

## Minimal Config

```yaml
dsn: "postgres://user:password@localhost:5432/app?sslmode=disable"
driver: postgres
renderer: sqlx
out_dir: ./internal/entity
tables: []
exclude:
  - schema_migrations
  - goose_db_version
  - migrations
on_conflict: skip
decimal_strategy: string
json_strategy: rawmessage
json_case_strategy: snake
generate_field_refs: false
nullable_strategy: pointer
type_overrides: []
```

## Core Keys

### `dsn`

Database connection string.

Examples:

```yaml
dsn: "postgres://postgres:postgres@localhost:5432/app_db?sslmode=disable"
```

```yaml
dsn: "root:password@tcp(localhost:3306)/app_db?parseTime=true"
```

```yaml
dsn: "./app.db"
```

### `driver`

Supported values:

- `postgres`
- `mysql`
- `mariadb`
- `sqlite`

### `renderer`

Supported values:

- `plain`
- `sqlx`
- `gorm`

Pick based on how you consume the generated entity layer:

- `plain` for clean structs
- `sqlx` for raw SQL and `db` tags
- `gorm` for GORM-ready models

See [Renderer Guide](./RENDERERS.md) for detail.

### `out_dir`

Directory where generated entity files are written.

### `tables`

Optional allowlist of tables.

```yaml
tables:
  - users
  - roles
  - permissions
```

Leave it empty to generate for all discovered tables.

### `exclude`

Table denylist.

Useful for:

- schema migration tables
- meta tables
- tables you do not want in the generated entity layer

### `on_conflict`

Controls what happens when a target file already exists but is not recognized as a managed schemagen file.

Supported values:

- `skip`
- `error`
- `backup`
- `overwrite`

Recommended:

- `skip` for safety in shared repos
- `backup` when iterating locally and you want less manual cleanup

## Type Strategy Knobs

### `decimal_strategy`

Supported values:

- `float64`
- `string`

Recommended:

- `string` for most production schemas
- `float64` only when precision loss is acceptable

### `json_strategy`

Supported values:

- `bytes`
- `rawmessage`

Behavior:

- `bytes` -> `[]byte`
- `rawmessage` -> `json.RawMessage`

Recommended:

- `rawmessage` for most JSON-heavy application code
- `bytes` when you want a more neutral binary boundary

### `json_case_strategy`

Supported values:

- `snake`
- `camel`

This affects both:

- base entity fields
- generated relation fields

Recommended:

- `snake` for conventional REST payloads
- `camel` for frontend-first JS/TS contracts

### `nullable_strategy`

Supported values:

- `pointer`
- `sqlnull`

Behavior:

- `pointer` -> nullable scalars become `*T`
- `sqlnull` -> nullable supported scalars become `sql.Null*`

Recommended:

- `pointer` for `gorm`, API-oriented code, and most app models
- `sqlnull` for explicit `database/sql` semantics

### `generate_field_refs`

Supported values:

- `true`
- `false`

When enabled, schemagen generates grouped DB column references per entity:

```go
var UserField = struct {
	ID    string
	Email string
}{
	ID:    "id",
	Email: "email",
}
```

Useful for:

- raw SQL query building
- avoiding repeated string literals
- safer field usage in `sqlx` and `database/sql` code

## Type Overrides

Use `type_overrides` when default mapping is not enough.

Typical use cases:

- `uuid.UUID`
- `decimal.Decimal`
- explicit JSON type overrides
- table-specific domain types

Example:

```yaml
type_overrides:
  - db_type: uuid
    go_type: uuid.UUID
    imports:
      - github.com/google/uuid

  - table: orders
    column: amount
    go_type: decimal.Decimal
    imports:
      - github.com/shopspring/decimal
```

Matching priority is documented in [Data Type Reference](../DATATYPES.md).

## Recommended Baselines

### `sqlx`

```yaml
renderer: sqlx
decimal_strategy: string
json_strategy: rawmessage
json_case_strategy: snake
generate_field_refs: true
nullable_strategy: pointer
```

### `gorm`

```yaml
renderer: gorm
decimal_strategy: string
json_strategy: rawmessage
json_case_strategy: snake
generate_field_refs: true
nullable_strategy: pointer
```

### `plain`

```yaml
renderer: plain
decimal_strategy: string
json_strategy: rawmessage
json_case_strategy: snake
generate_field_refs: false
nullable_strategy: pointer
```

## Relations Config Path

The main config does not currently embed the relations path.

By default, schemagen resolves relations from:

1. `schemagen.relations/`
2. fallback legacy file `schemagen.relations.yaml`

You can override that at runtime:

```bash
schemagen generate --relations-config config/relations
```

See [Relations Guide](./RELATIONS.md).
