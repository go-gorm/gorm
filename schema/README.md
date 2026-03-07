# schema

This package is responsible for parsing Go structs into GORM's internal model representation used for query building and migration.

## Core types

| Type | File | Description |
|------|------|-------------|
| `Schema` | `schema.go` | Parsed representation of a model struct |
| `Field` | `field.go` | A single struct field with its column mapping and metadata |
| `Relationship` | `relationship.go` | Association between two schemas (HasOne, HasMany, BelongsTo, Many2Many) |
| `Index` | `index.go` | Index definition parsed from struct tags |
| `Constraint` | `constraint.go` | Database constraint (unique, foreign key, check) |

## How parsing works

1. `Parse(model, cacheStore, namer)` reflects on the struct type
2. Each field is mapped to a `Field` with column name, data type, tag options
3. Relationships are resolved by matching foreign keys across schemas
4. The result is cached in `cacheStore` (an LRU cache) to avoid repeated reflection

## Struct tag options

GORM reads the `gorm:` struct tag. Common options:

```go
type User struct {
    ID        uint   `gorm:"primaryKey"`
    Name      string `gorm:"column:full_name;size:100;not null"`
    Email     string `gorm:"uniqueIndex"`
    DeletedAt *time.Time `gorm:"index"`
}
```

See [GORM model documentation](https://gorm.io/docs/models.html) for the full tag reference.

## Interfaces

Implement these interfaces on a custom type to control GORM behaviour per field:

| Interface | Method | Purpose |
|-----------|--------|---------|
| `GormDataTypeInterface` | `GormDataType() string` | Custom column type |
| `CreateClausesInterface` | `CreateClauses(*Field)` | Extra clauses on insert |
| `QueryClausesInterface` | `QueryClauses(*Field)` | Extra clauses on select |
| `UpdateClausesInterface` | `UpdateClauses(*Field)` | Extra clauses on update |
| `DeleteClausesInterface` | `DeleteClauses(*Field)` | Extra clauses on delete |

## Naming strategies

The `Namer` interface (defined in `naming.go`) controls how struct and field names are translated to table and column names. The default strategy converts `CamelCase` to `snake_case` and pluralises table names.

```go
db, _ := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
    NamingStrategy: schema.NamingStrategy{
        TablePrefix:   "app_",
        SingularTable: true,
    },
})
```
