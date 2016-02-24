# Polymorphism

Supports polymorphic has-many and has-one associations.

```go
  type Cat struct {
    Id    int
    Name  string
    Toy   Toy `gorm:"polymorphic:Owner;"`
  }

  type Dog struct {
    Id   int
    Name string
    Toy  Toy `gorm:"polymorphic:Owner;"`
  }

  type Toy struct {
    Id        int
    Name      string
    OwnerId   int
    OwnerType string
  }
```

Note: polymorphic belongs-to and many-to-many are explicitly NOT supported, and will throw errors.
