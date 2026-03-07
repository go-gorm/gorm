# clause

This package provides the SQL clause building blocks used by GORM to construct queries.

## Core interfaces

- **`Interface`** — implemented by every clause; requires `Name()`, `Build()`, and `MergeClause()`
- **`Expression`** — a value that can write itself into a SQL builder (e.g. `Eq`, `IN`, `And`)
- **`Builder`** — the write target passed to `Build()`; wraps a `*Statement`

## Available clauses

| Clause | File | SQL keyword |
|--------|------|-------------|
| `Select` | `select.go` | `SELECT` |
| `From` | `from.go` | `FROM` |
| `Where` | `where.go` | `WHERE` |
| `Joins` | `joins.go` | `JOIN` |
| `GroupBy` | `group_by.go` | `GROUP BY` |
| `OrderBy` | `order_by.go` | `ORDER BY` |
| `Limit` | `limit.go` | `LIMIT` |
| `Insert` | `insert.go` | `INSERT INTO` |
| `Values` | `values.go` | `VALUES` |
| `OnConflict` | `on_conflict.go` | `ON CONFLICT` |
| `Update` | `update.go` | `UPDATE` |
| `Set` | `set.go` | `SET` |
| `Delete` | `delete.go` | `DELETE` |
| `With` | `with.go` | `WITH` (CTE) |
| `For` | `locking.go` | `FOR UPDATE / SHARE` |

## Expressions

Common expressions defined in `expression.go`:

| Expression | Example SQL |
|------------|-------------|
| `Eq{Col, Val}` | `col = ?` |
| `Neq{Col, Val}` | `col <> ?` |
| `Gt` / `Gte` / `Lt` / `Lte` | `col > ?` etc. |
| `In{Col, Values}` | `col IN (?,?)` |
| `Like{Col, Val}` | `col LIKE ?` |
| `And{...}` / `Or{...}` | `(a AND b)` |
| `Not{...}` | `NOT (a)` |
| `Expr{SQL, Vars}` | raw SQL fragment |

## Special constants

```go
clause.PrimaryKey   // references the primary key column
clause.CurrentTable // references the model's table
clause.Associations // used in association operations
```

## Example: building a WHERE clause manually

```go
db.Where(clause.And(
    clause.Eq{Column: "status", Value: "active"},
    clause.Gt{Column: "age", Value: 18},
)).Find(&users)
```

See [GORM query documentation](https://gorm.io/docs/query.html) for higher-level usage.
