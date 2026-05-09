# schemagen

[![CI](https://github.com/nurularifin27/schemagen/actions/workflows/ci.yml/badge.svg)](https://github.com/nurularifin27/schemagen/actions/workflows/ci.yml)
[![Latest Tag](https://img.shields.io/github/v/tag/nurularifin27/schemagen)](https://github.com/nurularifin27/schemagen/tags)
[![Go Version](https://img.shields.io/github/go-mod/go-version/nurularifin27/schemagen)](https://github.com/nurularifin27/schemagen/blob/main/go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/nurularifin27/schemagen)](https://goreportcard.com/report/github.com/nurularifin27/schemagen)
[![License](https://img.shields.io/github/license/nurularifin27/schemagen)](./LICENSE)

Schema-to-Go entity generator with safe regeneration, renderer-aware output, driver-specific type mapping, and explicit relation config.

`schemagen` is built for teams that want generated entities without giving up control:

- `plain` renderer for clean domain structs
- `sqlx` renderer for raw SQL / `database/sql` / `sqlx`
- `gorm` renderer for practical GORM-ready models
- managed regeneration that preserves manual code outside generated sections
- explicit type and relation override hooks when database metadata is not enough

## Why

Most schema generators fail in one of two ways:

- too naive, so the generated code is wrong or noisy
- too magical, so the output is hard to trust

`schemagen` stays in the middle:

- type mapping is driver-aware
- relation generation is explicit, not guessed blindly
- config is small for simple schemas and still scalable for large ones
- generated files can be regenerated safely without destroying manual code

## Install

```bash
go install github.com/nurularifin27/schemagen/cmd/schemagen@latest
```

Check the installed version:

```bash
schemagen --version
schemagen -v
schemagen version
```

## Quick Start

Initialize config:

```bash
schemagen init
```

That writes:

```text
schemagen.yaml
schemagen.relations/default.yaml
```

Generate entities:

```bash
schemagen generate --config schemagen.yaml
```

The root command stays backward-compatible, so this also works:

```bash
schemagen --config schemagen.yaml
```

## Default Project Layout

```text
.
├── schemagen.yaml
├── schemagen.relations/
│   └── default.yaml
└── internal/
    └── entity/
```

For larger schemas, split relations by domain:

```text
schemagen.relations/
  auth.yaml
  org.yaml
  catalog.yaml
  inventory.yaml
  sales.yaml
  menu.yaml
```

## Choose a Renderer

### `plain`

Use this when you want:

- clean structs with minimal DB coupling
- domain model generation
- manual query / repository mapping outside the generated layer

### `sqlx`

Use this when you want:

- `db:"..."` tags
- `database/sql` or `sqlx`
- raw SQL with safer field references and consistent entity structs

This is the most practical renderer today for query-heavy backends.

### `gorm`

Use this when you want:

- GORM-ready structs
- `TableName()` generation
- GORM relation fields
- `deleted_at` mapped to `gorm.DeletedAt`

The GORM renderer is intentionally practical, not a full schema-to-tag mirror.

## Core Features

- renderer-based output: `plain`, `sqlx`, `gorm`
- driver-specific type mapping for `postgres`, `mysql`, `mariadb`, and `sqlite`
- configurable strategies for decimal, JSON, JSON tag casing, nullable handling
- optional generated field references like `UserField.ID`
- explicit `type_overrides` for custom DB and domain types
- explicit relation config with grouped format
- directory-based relations config for large schemas
- safe regeneration with managed sections
- conflict handling for unmanaged files
- concise CLI logging with `--verbose` and `--quiet`

## Documentation

Start here:

- [Cookbook](./docs/COOKBOOK.md)
- [Configuration Guide](./docs/CONFIGURATION.md)
- [Renderer Guide](./docs/RENDERERS.md)
- [Relations Guide](./docs/RELATIONS.md)
- [Workflow Guide](./docs/WORKFLOW.md)
- [Data Type Reference](./DATATYPES.md)

## CLI Reference

Commands:

- `schemagen`
- `schemagen generate`
- `schemagen init`
- `schemagen version`
- `schemagen completion`

`schemagen` behaves like `schemagen generate` and accepts the same generation flags.

### `schemagen generate`

| Flag | Default | Values | Purpose |
| --- | --- | --- | --- |
| `--config` | `schemagen.yaml` | path | Main config file |
| `--relations-config` | `schemagen.relations` | path | Relations config file or directory |
| `--dsn` | none | DSN string | Database connection string |
| `--driver` | from config | `postgres`, `mysql`, `mariadb`, `sqlite` | Database driver |
| `--renderer` | from config | `plain`, `sqlx`, `gorm` | Output renderer |
| `--out-dir` | from config | path | Output directory |
| `--tables` | all tables | comma-separated list | Include only selected tables |
| `--exclude` | from config | comma-separated list | Exclude selected tables |
| `--on-conflict` | from config | `skip`, `error`, `backup`, `overwrite` | Existing unmanaged file policy |
| `--decimal-strategy` | from config | `float64`, `string` | Decimal mapping strategy |
| `--json-strategy` | from config | `bytes`, `rawmessage` | JSON mapping strategy |
| `--json-case-strategy` | from config | `snake`, `camel` | JSON tag naming strategy |
| `--generate-field-refs` | from config | bool | Generate grouped field references like `UserField.ID` |
| `--nullable-strategy` | from config | `pointer`, `sqlnull` | Nullable scalar mapping strategy |
| `--verbose` | `false` | bool | Print per-table generation details |
| `--quiet` | `false` | bool | Suppress informational output |

Examples:

```bash
schemagen generate --config schemagen.yaml

schemagen generate \
  --config schemagen.yaml \
  --relations-config schemagen.relations

schemagen generate \
  --dsn "postgres://user:pass@localhost:5432/app?sslmode=disable" \
  --driver postgres \
  --renderer sqlx \
  --out-dir ./internal/entity

schemagen generate \
  --config schemagen.yaml \
  --renderer gorm \
  --decimal-strategy string \
  --json-strategy rawmessage \
  --json-case-strategy snake \
  --generate-field-refs \
  --nullable-strategy pointer \
  --on-conflict backup
```

### `schemagen init`

Behavior:

- writes `schemagen.yaml`
- writes `schemagen.relations/default.yaml`

| Flag | Default | Purpose |
| --- | --- | --- |
| `--path` | `schemagen.yaml` | Main config output path |
| `--relations-path` | `schemagen.relations` | Relations config output file or directory path |
| `--force` | `false` | Overwrite existing config files |

Examples:

```bash
schemagen init

schemagen init \
  --path config/schemagen.yaml \
  --relations-path config/schemagen.relations

schemagen init --force
```

### `schemagen completion`

```bash
schemagen completion zsh
schemagen completion bash
schemagen completion fish
schemagen completion powershell
```

## Logging Behavior

- `INFO` goes to normal output
- `WARN` goes to error output
- `ERROR` is returned once by the CLI
- default mode prints summary plus warnings
- `--verbose` prints per-table progress
- `--quiet` suppresses informational output

Example summary:

```text
INFO  generated=12 skipped=1 backed_up=0 overwritten=0 tables=13 renderer=sqlx out_dir=./internal/entity
```

## Config Example

Recommended baseline for many real projects:

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
generate_field_refs: true
nullable_strategy: pointer
type_overrides: []
```

See [Configuration Guide](./docs/CONFIGURATION.md) for full detail.

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
