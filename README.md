# schemagen

[![CI](https://github.com/nurularifin27/schemagen/actions/workflows/ci.yml/badge.svg)](https://github.com/nurularifin27/schemagen/actions/workflows/ci.yml)
[![Latest Tag](https://img.shields.io/github/v/tag/nurularifin27/schemagen)](https://github.com/nurularifin27/schemagen/tags)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nurularifin27/schemagen)](https://github.com/nurularifin27/schemagen/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/nurularifin27/schemagen)](https://goreportcard.com/report/github.com/nurularifin27/schemagen)

Schema-to-Go entity generator with safe regeneration and manual code preservation.

## Features

- Generate Go entity structs from database schema
- Multiple renderers: `plain`, `sqlx`, `gorm`
- Generic scan-type-based field inference
- Preserve manual code outside managed `SECTION` markers
- Merge generated imports with manual imports without duplicates
- Handle unmanaged file conflicts with `skip`, `error`, `backup`, or `overwrite`
- Cobra-based CLI with shell completion support

## Install

```bash
go install github.com/nurularifin27/schemagen/cmd/schemagen@latest
```

## Quick Start

Initialize config:

```bash
schemagen init
```

Generate entities from that config:

```bash
schemagen generate --config schemagen.yaml
```

Root command stays backward compatible, so this also works:

```bash
schemagen --config schemagen.yaml
```

## Run Without Install

```bash
go run ./cmd/schemagen init
go run ./cmd/schemagen generate --config schemagen.yaml
go run ./cmd/schemagen generate --config schemagen.yaml --renderer gorm
go run ./cmd/schemagen generate --config schemagen.yaml --relations-config schemagen.relations.yaml
```

## Build

```bash
go build ./cmd/schemagen
```

## Completion

```bash
schemagen completion zsh
schemagen completion bash
schemagen completion fish
schemagen completion powershell
```

## CLI Reference

Commands:

- `schemagen`
- `schemagen generate`
- `schemagen init`
- `schemagen completion`

`schemagen` behaves like `schemagen generate` and accepts the same generation flags.

### `schemagen generate`

Flags:

| Flag | Default | Values | Purpose |
| --- | --- | --- | --- |
| `--config` | `schemagen.yaml` | path | Main config file |
| `--relations-config` | `schemagen.relations.yaml` | path | Relations config file |
| `--dsn` | none | DSN string | Database connection string |
| `--driver` | from config | `postgres`, `mysql`, `mariadb`, `sqlite` | Database driver |
| `--renderer` | from config | `plain`, `sqlx`, `gorm` | Output renderer |
| `--out-dir` | from config | path | Output directory |
| `--tables` | all tables | comma-separated list | Include only selected tables |
| `--exclude` | from config | comma-separated list | Exclude selected tables |
| `--on-conflict` | from config | `skip`, `error`, `backup`, `overwrite` | Existing unmanaged file policy |
| `--decimal-strategy` | from config | `float64`, `string` | Decimal mapping strategy |
| `--json-strategy` | from config | `bytes`, `rawmessage` | JSON mapping strategy |
| `--nullable-strategy` | from config | `pointer`, `sqlnull` | Nullable scalar mapping strategy |
| `--verbose` | `false` | bool | Print per-table generation details |
| `--quiet` | `false` | bool | Suppress informational output |

Examples:

```bash
schemagen generate --config schemagen.yaml

schemagen generate \
  --config schemagen.yaml \
  --relations-config schemagen.relations.yaml

schemagen generate \
  --dsn "postgres://user:pass@localhost:5432/app?sslmode=disable" \
  --driver postgres \
  --renderer sqlx \
  --out-dir ./internal/entity

schemagen generate \
  --config schemagen.yaml \
  --tables users,orders,order_items

schemagen generate \
  --config schemagen.yaml \
  --renderer gorm \
  --decimal-strategy string \
  --json-strategy rawmessage \
  --nullable-strategy pointer \
  --on-conflict backup

schemagen generate \
  --config schemagen.yaml \
  --verbose

schemagen generate \
  --config schemagen.yaml \
  --quiet
```

CLI output behavior:

- `INFO` lines go to normal command output
- `WARN` lines go to error output
- `ERROR` lines are returned by the command and printed once by the CLI entrypoint
- default mode prints high-signal info plus warnings
- `--verbose` prints per-table progress
- `--quiet` suppresses informational output

Example default output:

```text
INFO  generated=12 skipped=1 backed_up=0 overwritten=0 tables=13 renderer=sqlx out_dir=./internal/entity
```

Example warning output:

```text
WARN  skip unmanaged file for table users: ./internal/entity/user.go
```

### `schemagen init`

Behavior:

- writes `schemagen.yaml`
- writes `schemagen.relations.yaml`

Flags:

| Flag | Default | Purpose |
| --- | --- | --- |
| `--path` | `schemagen.yaml` | Main config output path |
| `--relations-path` | `schemagen.relations.yaml` | Relations config output path |
| `--force` | `false` | Overwrite existing config files |

Examples:

```bash
schemagen init

schemagen init \
  --path config/schemagen.yaml \
  --relations-path config/schemagen.relations.yaml

schemagen init --force
```

### `schemagen completion`

Examples:

```bash
schemagen completion zsh
schemagen completion bash
schemagen completion fish
schemagen completion powershell
```

## Test

```bash
go test ./cmd/schemagen ./dbtype ./entitygen
```

Integration tests:

```bash
SCHEMAGEN_TEST_POSTGRES_DSN='postgres://user:pass@127.0.0.1:5432/app?sslmode=disable' \
SCHEMAGEN_TEST_MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/app?parseTime=true' \
SCHEMAGEN_TEST_MARIADB_DSN='user:pass@tcp(127.0.0.1:3307)/app?parseTime=true' \
go test ./entitygen
```

If a DSN is missing, the matching integration test is skipped.

## Config

`schemagen init` writes a real `schemagen.yaml`. There is no separate example file because the CLI reads `schemagen.yaml` by default.

Default config:

```yaml
dsn: ""
driver: postgres
renderer: sqlx
out_dir: ./internal/entity
tables: []
exclude:
  - schema_migrations
  - goose_db_version
  - migrations
on_conflict: skip
decimal_strategy: float64
json_strategy: bytes
nullable_strategy: pointer
type_overrides: []
```

Conflict policies:

- `skip`: leave unmanaged files untouched and warn
- `error`: stop when an unmanaged file already exists
- `backup`: move unmanaged file to `*.bak.<timestamp>` and write a new generated file
- `overwrite`: replace unmanaged file directly

Type mapping strategies:

- `renderer`: `plain`, `sqlx`, or `gorm`
- `decimal_strategy`: `float64` or `string`
- `json_strategy`: `bytes` (`[]byte`) or `rawmessage` (`json.RawMessage`)
- `nullable_strategy`: `pointer` or `sqlnull`
- `type_overrides`: explicit overrides for driver gaps, custom DB types, or per-column mapping

Renderer behavior:

- `plain`: no struct tags
- `sqlx`: `db` + `json` tags
- `gorm`: `gorm` + `json` tags and `TableName()` method

Renderer examples:

`plain`

```go
type User struct {
	ID        *int64
	Email     string
	CreatedAt *time.Time

	Profile *Profile `json:"profile,omitempty"`
}
```

`sqlx`

```go
type User struct {
	ID        *int64     `db:"id" json:"id"`
	Email     string     `db:"email" json:"email"`
	CreatedAt *time.Time `db:"created_at" json:"created_at"`

	Profile *Profile `json:"profile,omitempty"`
}
```

`gorm`

```go
type User struct {
	ID        *int64     `gorm:"column:id;primaryKey" json:"id"`
	Email     string     `gorm:"column:email;not null" json:"email"`
	CreatedAt *time.Time `gorm:"column:created_at" json:"created_at"`

	Profile *Profile `gorm:"foreignKey:UserID;references:ID" json:"profile,omitempty"`
}

func (*User) TableName() string {
	return TableNameUser
}
```

Nullable behavior:

- `pointer`: nullable scalars become `*T`
- `sqlnull`: nullable scalars use `database/sql` null types when available

Nullable strategy details:

- `plain`
  - `pointer`: best for domain models with minimal DB coupling
  - `sqlnull`: only useful if you explicitly want DB null boundary types in plain structs
- `sqlx`
  - `pointer`: pragmatic default for API-facing or service-facing structs
  - `sqlnull`: good when you want explicit scan semantics from `database/sql`
- `gorm`
  - `pointer`: recommended default
  - `sqlnull`: supported for scalar nullable types, but generally less idiomatic than pointers in GORM models

## Relations Config

Use a separate `schemagen.relations.yaml` file to keep relation mapping out of the main config.

Example:

```yaml
relations:
  - table: orders
    kind: belongs_to
    target_table: users
    foreign_key: user_id
    target_key: id

  - table: users
    kind: has_many
    field: Orders
    target_table: orders
    foreign_key: user_id
    target_key: id

  - table: users
    kind: has_one
    field: Profile
    target_table: user_profiles
    foreign_key: user_id
    target_key: id

  - table: users
    kind: many_to_many
    field: Roles
    target_table: roles
    join_table: user_roles
    join_foreign_key: user_id
    join_target_key: role_id
    source_key: id
    target_key: id

  - table: users
    kind: many_to_many
    field: Roles
    pivot_field: UserRoles
    target_table: roles
    join_table: user_roles
    join_foreign_key: user_id
    join_target_key: role_id
    source_key: id
    target_key: id
```

Supported kinds:

- `belongs_to`
- `has_one`
- `has_many`
- `many_to_many`

For `many_to_many`, `pivot_field` is optional. Use it when your join table is a real pivot entity with extra columns and you want both:

- direct access to the target collection, for example `Roles []*Role`
- explicit access to the pivot rows, for example `UserRoles []*UserRole`

`field` is optional. If it is omitted, schemagen derives a default:

- `belongs_to`, `has_one` -> singular target struct name
- `has_many`, `many_to_many` -> plural target struct name

Examples:

- `target_table: users` + `belongs_to` -> `User`
- `target_table: order_items` + `has_many` -> `OrderItems`

Generated relation field defaults:

- `belongs_to`, `has_one` -> `*Target`
- `has_many`, `many_to_many` -> `[]*Target`

Renderer behavior for relation fields:

- `plain` and `sqlx`: relation fields only get `json:",omitempty"`
- `gorm`: relation fields get minimal `gorm` relation metadata plus `json:",omitempty"`

Recommended relation patterns:

- Dumb join table:
  keep a single `many_to_many` relation and skip `pivot_field`
- Rich pivot table with payload columns:
  use `many_to_many` plus `pivot_field`, and also define `belongs_to` relations on the pivot table itself

Example rich pivot config:

```yaml
relations:
  - table: users
    kind: many_to_many
    field: Roles
    pivot_field: UserRoles
    target_table: roles
    join_table: user_roles
    join_foreign_key: user_id
    join_target_key: role_id
    source_key: id
    target_key: id

  - table: user_roles
    kind: belongs_to
    field: User
    target_table: users
    foreign_key: user_id
    target_key: id

  - table: user_roles
    kind: belongs_to
    field: Role
    target_table: roles
    foreign_key: role_id
    target_key: id
```

Result:

- `User.Roles []*Role` gives direct many-to-many navigation
- `User.UserRoles []*UserRole` exposes pivot rows and payload columns
- `UserRole.User` and `UserRole.Role` let you traverse the pivot entity explicitly

Relation examples by renderer:

`plain` / `sqlx`

```go
type Order struct {
	ID     *int64 `db:"id" json:"id"`
	UserID int64  `db:"user_id" json:"user_id"`

	User *User `json:"user,omitempty"`
}
```

`gorm`

```go
type Order struct {
	ID     *int64 `gorm:"column:id;primaryKey" json:"id"`
	UserID int64  `gorm:"column:user_id;not null" json:"user_id"`

	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}
```

Current relation model is explicit-config-first. Foreign key inference from database metadata is not implemented yet.

Example overrides:

```yaml
type_overrides:
  - db_type: uuid
    go_type: uuid.UUID
    imports:
      - github.com/google/uuid
  - table: users
    column: payload
    go_type: json.RawMessage
    imports:
      - encoding/json
```

## Type Compatibility

Current support level by driver:

| Area | Postgres | MySQL | MariaDB | SQLite |
| --- | --- | --- | --- | --- |
| Core integers / bool / text / time | Strong | Strong | Strong | Strong |
| Unsigned integers | N/A | Strong | Strong | N/A |
| `decimal` / `numeric` strategies | Strong | Strong | Strong | Strong |
| `json` / `jsonb` strategies | Strong | Strong | Strong | Strong |
| UUID default mapping | Strong | Fallback | Fallback | Fallback |
| Basic array mapping | Partial | N/A | N/A | N/A |
| Enum / domain / custom DB types | Override-driven | Override-driven | Override-driven | Override-driven |

Interpretation:

- `Strong`: explicit built-in mapping exists.
- `Partial`: some common cases are built in, but not the full database type surface.
- `Fallback`: use `type_overrides` if you need a precise Go type.

For production schemas with domain-specific types, prefer `type_overrides` instead of depending on implicit fallback behavior.

## Manual Code Policy

Generated files are editable. `schemagen` only manages code inside its section markers.

- Manual methods, getters, setters, and helpers outside markers are preserved
- Manual relations below the managed base section are preserved
- Manual imports are preserved and merged with generated imports
