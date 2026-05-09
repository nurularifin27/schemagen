# Relations Guide

`schemagen` uses explicit relation config. It does not currently infer relations from foreign keys automatically.

That is intentional.

Blind inference tends to produce wrong relation names, wrong cardinality, and noisy output on non-trivial schemas. The current model is explicit-first and predictable.

## Default Resolution

By default, schemagen loads relations from:

1. `schemagen.relations/`
2. fallback legacy file `schemagen.relations.yaml`

If you want a different path:

```bash
schemagen generate --relations-config config/relations
```

The path may be:

- a single YAML file
- a directory of YAML files

When a directory is used, schemagen merges all `*.yaml` and `*.yml` files in lexical order.

## Recommended Directory Layout

For small schemas:

```text
schemagen.relations/
  default.yaml
```

For medium or large schemas:

```text
schemagen.relations/
  auth.yaml
  org.yaml
  catalog.yaml
  inventory.yaml
  sales.yaml
  menu.yaml
```

## Recommended Authoring Rule

Do not split one table across many files unless you have a very strong reason.

Better rule:

- each table has one owning domain file
- other files may reference that table in `target_table`
- but the table's own generated relation block should live in one place

This reduces maintenance and avoids silent ownership drift.

## Config Format

Recommended grouped format:

```yaml
tables:
  orders:
    relations:
      - kind: belongs_to
        target_table: users
        foreign_key: user_id
        target_key: id

  users:
    relations:
      - kind: has_many
        target_table: orders
        foreign_key: user_id
        target_key: id
```

Legacy flat format is still supported:

```yaml
relations:
  - table: orders
    kind: belongs_to
    target_table: users
    foreign_key: user_id
    target_key: id
```

## Supported Kinds

- `belongs_to`
- `has_one`
- `has_many`
- `many_to_many`

## Cardinality Guidance

### `belongs_to`

Use when the current table carries the foreign key.

Example:

```yaml
tables:
  orders:
    relations:
      - kind: belongs_to
        target_table: users
        foreign_key: user_id
        target_key: id
```

### `has_one`

Use when the target table points back to the current table with one-to-one semantics.

Example:

```yaml
tables:
  users:
    relations:
      - kind: has_one
        target_table: user_profiles
        foreign_key: user_id
        target_key: id
```

### `has_many`

Use when many target rows point back to one source row.

Example:

```yaml
tables:
  users:
    relations:
      - kind: has_many
        target_table: orders
        foreign_key: user_id
        target_key: id
```

### `many_to_many`

Use only for pure join tables.

Example:

```yaml
tables:
  users:
    relations:
      - kind: many_to_many
        target_table: roles
        join_table: user_roles
        join_foreign_key: user_id
        join_target_key: role_id
        source_key: id
        target_key: id
```

Important:

- do not use `many_to_many` for join tables that carry business payload
- if the join table contains real domain columns, model it as a normal entity instead

## Field Naming

`field` is optional.

If omitted, schemagen derives a default:

- `belongs_to` -> singular target struct name
- `has_one` -> singular target struct name
- `has_many` -> plural target struct name
- `many_to_many` -> plural target struct name

Examples:

- `target_table: users` + `belongs_to` -> `User`
- `target_table: order_items` + `has_many` -> `OrderItems`

Use explicit `field` when:

- multiple FKs target the same table
- a default name would be ambiguous
- you want domain-specific naming

Example:

```yaml
tables:
  stock_adjustments:
    relations:
      - kind: belongs_to
        field: UserCreate
        target_table: users
        foreign_key: created_by
        target_key: id

      - kind: belongs_to
        field: UserApprove
        target_table: users
        foreign_key: approved_by
        target_key: id
```

## Large Schema Advice

For large schemas:

- keep grouped format
- split files by domain
- keep one table owned by one file
- use explicit names for multi-FK cases
- avoid giant single-file relation config

## Duplicate Definitions

Directory-based relation loading now rejects duplicate relation definitions.

That is intentional.

If the same relation is declared twice across files, schemagen fails instead of silently merging conflicting state.
