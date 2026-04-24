# schemagen

Schema-to-Go entity generator with safe regeneration and manual code preservation.

## Features

- Generate Go entity structs from database schema
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

## Test

```bash
go test ./cmd/schemagen ./dbtype ./entitygen
```

## Config

`schemagen init` writes a real `schemagen.yaml`. There is no separate example file because the CLI reads `schemagen.yaml` by default.

Default config:

```yaml
dsn: ""
driver: postgres
out_dir: ./internal/entity
tables: []
exclude:
  - schema_migrations
  - goose_db_version
  - migrations
on_conflict: skip
```

Conflict policies:

- `skip`: leave unmanaged files untouched and warn
- `error`: stop when an unmanaged file already exists
- `backup`: move unmanaged file to `*.bak.<timestamp>` and write a new generated file
- `overwrite`: replace unmanaged file directly

## Manual Code Policy

Generated files are editable. `schemagen` only manages code inside its section markers.

- Manual methods, getters, setters, and helpers outside markers are preserved
- Manual relations below the managed base section are preserved
- Manual imports are preserved and merged with generated imports
