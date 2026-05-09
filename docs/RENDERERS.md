# Renderer Guide

`schemagen` supports three output renderers:

- `plain`
- `sqlx`
- `gorm`

They all use the same schema introspection and type mapping layer, but the generated struct surface is different.

## `plain`

### What it generates

- clean structs
- `json` tags
- no `db` tag
- no `gorm` tag

### Best for

- domain/entity layers
- service-layer models
- custom repository mapping
- projects that do not want DB-library-specific tags in the entity layer

### Example

```go
type User struct {
	ID        int64      `json:"id"`
	Email     string     `json:"email"`
	DeletedAt *time.Time `json:"deleted_at"`
}
```

### Trade-off

You get clean output, but less out-of-the-box convenience for `sqlx` and GORM.

## `sqlx`

### What it generates

- `db:"column_name"`
- `json:"..."`
- optional relation fields
- optional field refs like `UserField.Email`

### Best for

- `database/sql`
- `sqlx`
- raw query repositories
- teams that want explicit SQL and consistent generated models

### Example

```go
type User struct {
	ID        int64      `db:"id" json:"id"`
	Email     string     `db:"email" json:"email"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at"`
}
```

### Why this renderer is strong

This renderer currently gives the best balance between:

- explicit SQL control
- safe field naming
- low framework coupling
- practical developer ergonomics

### Typical settings

```yaml
renderer: sqlx
generate_field_refs: true
nullable_strategy: pointer
json_case_strategy: snake
```

## `gorm`

### What it generates

- `gorm:"..."`
- `json:"..."`
- `TableName()` method
- relation fields
- `gorm.DeletedAt` for `deleted_at`

### Best for

- GORM application models
- preload-heavy code
- CRUD-centric backends that want direct GORM model usage

### Example

```go
type User struct {
	ID        int64          `gorm:"column:id;primaryKey" json:"id"`
	Email     string         `gorm:"column:email;not null" json:"email"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at"`
}

func (*User) TableName() string {
	return TableNameUser
}
```

### Current scope

The GORM renderer is designed to be:

- safe
- idiomatic for common usage
- not overloaded with every possible schema detail

It supports:

- standard column tags
- relation fields from explicit relations config
- practical soft delete handling

It does not try to become a full schema mirror for every GORM tag permutation.

## Relation Fields Across Renderers

Supported relation kinds:

- `belongs_to`
- `has_one`
- `has_many`
- `many_to_many`

Important:

- `many_to_many` is intended only for pure join tables
- if a join table carries business payload, treat it as a normal entity instead

### Generated relation shapes

- `belongs_to` -> `*Target`
- `has_one` -> `*Target`
- `has_many` -> `[]*Target`
- `many_to_many` -> `[]*Target`

## Which Renderer Should You Pick?

### Pick `plain` if

- you want clean structs first
- DB integration convenience is secondary

### Pick `sqlx` if

- your team writes raw SQL
- you want strong control with low abstraction cost
- you want generated structs to help query code directly

### Pick `gorm` if

- GORM is already the real persistence layer
- you want generated models that are directly usable

## Practical Recommendation

For many backends:

- start with `sqlx`
- use `gorm` only if the repo is truly GORM-centric
- use `plain` when the entity layer should stay DB-neutral
