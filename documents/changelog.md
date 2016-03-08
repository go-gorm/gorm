# Change Log

## v1.0

#### Breaking Changes

* **`gorm.Open` return type `*gorm.DB` instead of `gorm.DB`**

* **Updating will only update changed fields**

  Most applications won't be affected, only when you are changing updating values in callbacks like `BeforeSave`, `BeforeUpdate`, you should use `scope.SetColumn` then, for example:

  ```go
  func (user *User) BeforeUpdate(scope *gorm.Scope) {
    if pw, err := bcrypt.GenerateFromPassword(user.Password, 0); err == nil {
      scope.SetColumn("EncryptedPassword", pw)
      // user.EncryptedPassword = pw  // doesn't work, won't including EncryptedPassword field when updating
    }
  }
  ```

* **Soft Delete's default querying scope will only check `deleted_at IS NULL`**

  Before it will check `deleted_at` less than `0001-01-02` also to exclude blank time, like:

  `SELECT * FROM users WHERE deleted_at IS NULL OR deleted_at <= '0001-01-02'`

  But it is not necessary if you are using type `*time.Time` for your model's `DeletedAt`, which has been used by `gorm.Model`, so below SQl is enough

  `SELECT * FROM users WHERE deleted_at IS NULL`

  So if you are using `gorm.Model`, then you are good, nothing need to be change, just make sure all records having blank time for `deleted_at` set to `NULL`, sample migrate script:

```go
import (
    "github.com/jinzhu/now"
)

func main() {
  var models = []interface{}{&User{}, &Image{}}
  for _, model := range models {
    db.Unscoped().Model(model).Where("deleted_at < ?", now.MustParse("0001-01-02")).Update("deleted_at", gorm.Expr("NULL"))
  }
}
```

* **New ToDBName logic**

  Before when GORM convert Struct, Field's name to db name, only those common initialisms from [golint](https://github.com/golang/lint/blob/master/lint.go#L702) like `HTTP`, `URI` are special handled.

  So field `HTTP`'s db name will be `http` not `h_t_t_p`, but some other initialisms like `SKU` that not in golint, it's db name will be `s_k_u`, which looks ugly, this release fixed this, any upper case initialisms should be converted correctly.

  If your applications using some upper case initialisms which doesn't exist in [golint](https://github.com/golang/lint/blob/master/lint.go#L702), you need to overwrite default column name with tag, like `gorm:"column:s_k_u"`, or alert your database's column name according to new logic

* Error `RecordNotFound` has been renamed to `ErrRecordNotFound`

* `mssql` driver has been moved out from default drivers, import it with `import _ "github.com/jinzhu/gorm/dialects/mssql"`

* `Hstore` has been moved to package `github.com/jinzhu/gorm/dialects/postgres`
