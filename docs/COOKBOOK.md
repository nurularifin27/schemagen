# Cookbook

This guide is for practical usage patterns, not theory.

If you want a fast starting point, copy the closest recipe and adjust from there.

## 1. Start a New `sqlx` Project

Recommended when:

- your app uses `database/sql` or `sqlx`
- you want explicit SQL
- you want generated structs and DB field references

Config:

```yaml
dsn: "postgres://postgres:postgres@localhost:5432/app_db?sslmode=disable"
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

Generate:

```bash
schemagen generate --config schemagen.yaml
```

Expected usage:

```go
cols := []string{
	entity.UserField.ID,
	entity.UserField.Email,
}
```

## 2. Start a New GORM Project

Recommended when:

- your app is already GORM-first
- you want direct model usage
- you want `deleted_at` handled idiomatically

Config:

```yaml
dsn: "postgres://postgres:postgres@localhost:5432/app_db?sslmode=disable"
driver: postgres
renderer: gorm
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

Notes:

- `deleted_at` becomes `gorm.DeletedAt`
- relation fields come from explicit relations config
- this is intentionally practical GORM output, not a full schema tag dump

## 3. Start a New `plain` Project

Recommended when:

- you want clean structs first
- the entity layer should stay DB-library-neutral
- repository mapping lives elsewhere

Config:

```yaml
dsn: "postgres://postgres:postgres@localhost:5432/app_db?sslmode=disable"
driver: postgres
renderer: plain
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

## 4. Use Folder-Based Relations Config

Recommended when:

- the schema is large
- one giant relations file is hard to maintain
- ownership should be split by domain

Layout:

```text
schemagen.relations/
  auth.yaml
  org.yaml
  catalog.yaml
  inventory.yaml
  sales.yaml
  menu.yaml
```

Each file should own the tables for its own domain:

```yaml
tables:
  users:
    relations:
      - kind: has_many
        target_table: sales_orders
        foreign_key: user_id
        target_key: id
```

Command:

```bash
schemagen generate --config schemagen.yaml
```

You do not need `--relations-config` if you use the default `schemagen.relations/` path.

## 5. Override UUID to `uuid.UUID`

Config:

```yaml
type_overrides:
  - db_type: uuid
    go_type: uuid.UUID
    imports:
      - github.com/google/uuid
```

Use this when:

- Postgres columns are `uuid`
- you want strong UUID types instead of plain `string`

## 6. Override Decimal to `decimal.Decimal`

Config:

```yaml
type_overrides:
  - db_type: decimal
    go_type: decimal.Decimal
    imports:
      - github.com/shopspring/decimal

  - db_type: numeric
    go_type: decimal.Decimal
    imports:
      - github.com/shopspring/decimal
```

Use this when:

- money, tax, price, or balance fields must stay lossless

Recommended baseline:

```yaml
decimal_strategy: string
```

Then add overrides only where domain precision really matters.

## 7. Override JSON to `json.RawMessage`

Global:

```yaml
json_strategy: rawmessage
```

Column-specific:

```yaml
type_overrides:
  - table: billing_records
    column: payload
    go_type: json.RawMessage
    imports:
      - encoding/json
```

Use the column-specific form when:

- MariaDB metadata exposes JSON as text-like
- only one or two JSON columns need explicit handling

## 8. Model a Simple `belongs_to`

Relations file:

```yaml
tables:
  orders:
    relations:
      - kind: belongs_to
        target_table: users
        foreign_key: user_id
        target_key: id
```

Generated relation:

```go
User *User
```

## 9. Model a `has_one`

Relations file:

```yaml
tables:
  users:
    relations:
      - kind: has_one
        target_table: user_profiles
        foreign_key: user_id
        target_key: id
```

Generated relation:

```go
Profile *UserProfile
```

## 10. Model a `has_many`

Relations file:

```yaml
tables:
  users:
    relations:
      - kind: has_many
        target_table: orders
        foreign_key: user_id
        target_key: id
```

Generated relation:

```go
Orders []*Order
```

## 11. Model a Pure `many_to_many`

Relations file:

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

Generated relation:

```go
Roles []*Role
```

Important:

- only use this for pure join tables
- if the join table carries business payload, do not model it as `many_to_many`

## 12. Model a Join Table With Payload

Do not do this:

```yaml
kind: many_to_many
```

if the join table has business columns like:

- `status`
- `assigned_at`
- `notes`
- `metadata`

Treat it as a normal entity instead.

Example:

```yaml
tables:
  users:
    relations:
      - kind: has_many
        field: UserRoles
        target_table: user_roles
        foreign_key: user_id
        target_key: id

  user_roles:
    relations:
      - kind: belongs_to
        target_table: users
        foreign_key: user_id
        target_key: id

      - kind: belongs_to
        target_table: roles
        foreign_key: role_id
        target_key: id
```

## 13. Handle Multiple FKs to the Same Table

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

Use explicit `field` here. Do not rely on default naming for this case.

## 14. Generate Only a Small Table Set

Config:

```yaml
tables:
  - users
  - roles
  - permissions
```

Or via CLI:

```bash
schemagen generate --config schemagen.yaml --tables users,roles,permissions
```

Useful when:

- bootstrapping a subset first
- validating a new config change
- keeping diffs smaller during early adoption

## 15. Exclude Tables You Never Want

Config:

```yaml
exclude:
  - schema_migrations
  - goose_db_version
  - migrations
```

This is better than cleaning those files up after generation.

## 16. Use `backup` for Safer Local Iteration

Config:

```yaml
on_conflict: backup
```

Use this when:

- you are testing generator changes locally
- unmanaged target files may already exist
- you want an easier rollback than `overwrite`

## 17. Use `sqlnull` When Scan Semantics Matter

Config:

```yaml
nullable_strategy: sqlnull
```

Use this when:

- you want explicit `database/sql` null handling
- pointer semantics are not enough for your boundary layer

Typical result:

- `string` -> `sql.NullString`
- `int64` -> `sql.NullInt64`
- `time.Time` -> `sql.NullTime`

For GORM and most API-facing code, `pointer` is still the better default.

## 18. Migrate From Legacy `schemagen.relations.yaml`

Current loading order:

1. `schemagen.relations/`
2. fallback `schemagen.relations.yaml`

Migration path:

1. create `schemagen.relations/`
2. split the legacy file into domain files
3. keep one table owned by one file
4. verify the split
5. remove the old file

## 19. Run Without Installing the Binary

```bash
go run ./cmd/schemagen init
go run ./cmd/schemagen generate --config schemagen.yaml
```

Useful when:

- testing local `schemagen` changes
- pinning execution to the checked-out source

## 20. Use From Another Repo Against Local Schemagen Source

Example:

```bash
cd /path/to/your/app
go run /absolute/path/to/schemagen/cmd/schemagen generate --config schemagen.yaml
```

Useful when:

- you are upgrading a consumer repo before cutting a release
- you want to validate behavior against local unreleased changes
