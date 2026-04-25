# Data Type Reference

Reference for current driver-specific type mapping in `schemagen`.

This document is intended as a practical guide for:

- understanding the current generated Go type per driver
- choosing `decimal_strategy`, `json_strategy`, and `nullable_strategy`
- writing `type_overrides` with the correct `db_type` values

This is a reference for the current implementation, not an abstract SQL type catalog.

## Override Matching Rules

`type_overrides` matches against the normalized `db_type` value exposed by schemagen.

Important:

- `db_type` is matched against `column.DatabaseType` only
- matching is case-insensitive
- `full_type` is not matched by `type_overrides`
- match priority is:
  - `table + column + db_type`
  - `table + column`
  - `table + db_type`
  - `column + db_type`
  - `table`
  - `column`
  - `db_type`

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

## Strategy Knobs

Some mappings are intentionally configurable.

### `decimal_strategy`

- `float64`
  - `numeric` / `decimal` -> `float64`
- `string`
  - `numeric` / `decimal` -> `string`

### `json_strategy`

- `bytes`
  - JSON -> `[]byte`
- `rawmessage`
  - JSON -> `json.RawMessage`

### `nullable_strategy`

- `pointer`
  - nullable scalars become `*T`
- `sqlnull`
  - nullable supported scalars become `sql.Null*`

## Postgres

Common `db_type` values currently recognized:

| `db_type` | Generated Go type |
| --- | --- |
| `smallint`, `int2` | `int16` |
| `integer`, `int`, `int4`, `serial`, `serial4` | `int32` |
| `bigint`, `int8`, `bigserial`, `serial8` | `int64` |
| `real`, `float4` | `float32` |
| `double precision`, `float8` | `float64` |
| `numeric`, `decimal` | `float64` or `string` |
| `boolean`, `bool` | `bool` |
| `char`, `character`, `bpchar`, `varchar`, `character varying`, `text`, `citext`, `name` | `string` |
| `date`, `time`, `time without time zone`, `timetz`, `time with time zone`, `timestamp`, `timestamp without time zone`, `timestamptz`, `timestamp with time zone` | `time.Time` |
| `interval` | `string` |
| `bytea` | `[]byte` |
| `json`, `jsonb` | `[]byte` or `json.RawMessage` |
| `uuid` | `string` |
| `inet`, `cidr`, `macaddr`, `macaddr8`, `xml`, `money`, `tsvector`, `tsquery` | `string` |

Postgres array support:

| `db_type` | Generated Go type |
| --- | --- |
| `_text`, `_varchar`, `_bpchar`, `_citext`, `_uuid` | `[]string` |
| `_bool` | `[]bool` |
| `_int2` | `[]int16` |
| `_int4` | `[]int32` |
| `_int8` | `[]int64` |
| `_float4` | `[]float32` |
| `_float8`, `_numeric` | `[]float64` |
| `_bytea` | `[][]byte` |

Notes:

- `uuid` stays `string` by default. Override it if you want `uuid.UUID`.
- `numeric` / `decimal` should usually be overridden for financial domains.
- many Postgres-specific advanced types still fall back to `string`.

Recommended overrides:

```yaml
type_overrides:
  - db_type: uuid
    go_type: uuid.UUID
    imports:
      - github.com/google/uuid

  - db_type: numeric
    go_type: decimal.Decimal
    imports:
      - github.com/shopspring/decimal
```

## MySQL

Common `db_type` values currently recognized:

| `db_type` | Generated Go type |
| --- | --- |
| `tinyint` | `bool`, `int8`, or `uint8` |
| `smallint` | `int16` or `uint16` |
| `mediumint`, `int`, `integer` | `int32` or `uint32` |
| `bigint` | `int64` or `uint64` |
| `decimal`, `numeric` | `float64` or `string` |
| `float` | `float32` |
| `double`, `double precision`, `real` | `float64` |
| `bit` | `[]byte` |
| `bool`, `boolean` | `bool` |
| `char`, `varchar`, `tinytext`, `text`, `mediumtext`, `longtext`, `enum`, `set` | `string` |
| `date`, `datetime`, `timestamp` | `time.Time` |
| `time` | `string` |
| `year` | `int16` |
| `binary`, `varbinary`, `tinyblob`, `blob`, `mediumblob`, `longblob` | `[]byte` |
| `json` | `[]byte` or `json.RawMessage` |
| `geometry`, `point`, `linestring`, `polygon`, `multipoint`, `multilinestring`, `multipolygon`, `geometrycollection` | `[]byte` |

Notes:

- unsigned behavior depends on `full_type`, but overrides still match only the `db_type`
- `tinyint(1)` is treated as `bool`
- `bit` is currently coarse and maps to `[]byte`

Recommended overrides:

```yaml
type_overrides:
  - db_type: decimal
    go_type: decimal.Decimal
    imports:
      - github.com/shopspring/decimal

  - table: flags
    column: mask
    db_type: bit
    go_type: uint64
```

## MariaDB

MariaDB currently mostly follows MySQL mapping, with one important caveat around JSON.

Common `db_type` values currently recognized:

| `db_type` | Generated Go type |
| --- | --- |
| `tinyint` | `bool`, `int8`, or `uint8` |
| `smallint` | `int16` or `uint16` |
| `mediumint`, `int`, `integer` | `int32` or `uint32` |
| `bigint` | `int64` or `uint64` |
| `decimal`, `numeric` | `float64` or `string` |
| `float` | `float32` |
| `double`, `double precision`, `real` | `float64` |
| `bit` | `[]byte` |
| `bool`, `boolean` | `bool` |
| `char`, `varchar`, `tinytext`, `text`, `mediumtext`, `longtext`, `enum`, `set` | `string` |
| `date`, `datetime`, `timestamp` | `time.Time` |
| `time` | `string` |
| `year` | `int16` |
| `binary`, `varbinary`, `tinyblob`, `blob`, `mediumblob`, `longblob` | `[]byte` |
| `json` | `[]byte` or `json.RawMessage` when exposed as logical JSON |
| `geometry`, `point`, `linestring`, `polygon`, `multipoint`, `multilinestring`, `multipolygon`, `geometrycollection` | `[]byte` |

MariaDB JSON caveat:

- MariaDB may expose JSON columns as text-like metadata instead of logical `json`
- when that happens, `db_type` may not be `json`
- `json_strategy` then cannot help by itself
- use explicit per-table or per-column override when necessary

Recommended override pattern:

```yaml
type_overrides:
  - table: billing_records
    column: payload
    go_type: json.RawMessage
    imports:
      - encoding/json
```

## SQLite

Common `db_type` values currently recognized:

| `db_type` | Generated Go type |
| --- | --- |
| `integer`, `int`, `tinyint`, `smallint`, `mediumint`, `bigint`, `unsigned big int`, `int2`, `int8` | `int64` |
| `real`, `double`, `double precision`, `float` | `float64` |
| `numeric`, `decimal` | `float64` or `string` |
| `boolean` | `bool` |
| `text`, `character`, `varchar`, `varying character`, `nchar`, `native character`, `nvarchar`, `clob` | `string` |
| `blob` | `[]byte` |
| `date`, `datetime`, `timestamp` | `time.Time` |
| `json` | `[]byte` or `json.RawMessage` |

Notes:

- SQLite type affinity is loose, so scan type may override simple name-based expectations
- in ambiguous cases, generated types are only as reliable as the driver metadata

## Nullable Behavior

Nullable handling happens after driver mapping.

Examples:

| Base Go type | `pointer` | `sqlnull` |
| --- | --- | --- |
| `string` | `*string` | `sql.NullString` |
| `bool` | `*bool` | `sql.NullBool` |
| `int16` | `*int16` | `sql.NullInt16` |
| `int32` | `*int32` | `sql.NullInt32` |
| `int64` | `*int64` | `sql.NullInt64` |
| `float64` | `*float64` | `sql.NullFloat64` |
| `time.Time` | `*time.Time` | `sql.NullTime` |
| `[]byte` | `[]byte` | `[]byte` |

Notes:

- primary keys and auto-increment fields are not converted to `sql.Null*`
- slice types are not pointer-wrapped

## Practical Guidance

Use defaults when:

- schema is straightforward
- you do not need strong domain types
- you are optimizing for fast code generation

Use `type_overrides` when:

- you want `uuid.UUID`
- you want `decimal.Decimal`
- MariaDB JSON needs explicit handling
- a specific column should map to a domain type
- driver metadata is too weak or ambiguous

Recommended baseline:

```yaml
decimal_strategy: string
json_strategy: rawmessage
nullable_strategy: pointer
```

Then add targeted overrides only where the domain requires stricter types.
