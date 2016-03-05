# Change Log

## v1.0

#### Breaking Changes

* **`gorm.Open` return `*gorm.DB` instead of `gorm.DB`**

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

* **Soft delete's default querying scope will only check `deleted_at IS NULL`**

  Before `db.Find(&user)` will generate querying SQL if user has `DeletedAt` field

  `SELECT * FROM users WHERE deleted_at IS NULL OR deleted_at <= '0001-01-02'`

  Now won't include blank time check `<= '0001-01-02` anymore, will generat SQL like:

  `SELECT * FROM users WHERE deleted_at IS NULL`

  So your application's `DeletedAt` field should not use `time.Time` as data type, need to use pointer `*time.Time` or something like `NullTime`.
  If you are using `gorm.Model`, then you are good, nothing need to be change, just make sure all records using blank time for `deleted_at` has been set to NULL, sample migration script:

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

  So field `HTTP`'s db name will be `http` not `h_t_t_p`, but some other initialisms like `SKU` that not in golint, it's db name will be `s_k_u`, this release fixed this, any upper case initialisms should be converted correctly.

  If your applications using some upper case initialisms which doesn't exist in [golint](https://github.com/golang/lint/blob/master/lint.go#L702), you need to overwrite generated column name with tag, like `sql:"column:s_k_u"`, or alert your database's column name according to new logic

* **Builtin `Hstore` struct for postgres has been moved to `github.com/jinzhu/gorm/dialects/postgres`**
