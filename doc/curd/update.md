# Update

### Update All Fields

`Save` will include all fields when perform the Updating SQL, even it is not changed

```go
db.First(&user)

user.Name = "jinzhu 2"
user.Age = 100
db.Save(&user)

//// UPDATE users SET name='jinzhu 2', age=100, birthday='2016-01-01', updated_at = '2013-11-17 21:34:10' WHERE id=111;
```

### Update Changed Fields

If you only want to update changed Fields, you could use `Update`, `Updates`

```go
// Update single attribute if it is changed
db.Model(&user).Update("name", "hello")
//// UPDATE users SET name='hello', updated_at='2013-11-17 21:34:10' WHERE id=111;

// Update single attribute with combined conditions
db.Model(&user).Where("active = ?", true).Update("name", "hello")
//// UPDATE users SET name='hello', updated_at='2013-11-17 21:34:10' WHERE id=111 AND active=true;


// Update multiple attributes with `map`, will only update those changed fields
db.Model(&user).Updates(map[string]interface{}{"name": "hello", "age": 18, "actived": false})
//// UPDATE users SET name='hello', age=18, actived=false, updated_at='2013-11-17 21:34:10' WHERE id=111;

// Update multiple attributes with `struct`, will only update those changed & non blank fields
db.Model(&user).Updates(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18, updated_at = '2013-11-17 21:34:10' WHERE id = 111;

// WARNING when update with struct, GORM will only update those fields that with non blank value
// For below Update, nothing will be updated as "", 0, false are blank values of their types
db.Model(&user).Updates(User{Name: "", Age: 0, Actived: false})
```

### Update Selected Fields

If you only want to update or ignore some fields when updating, you could use `Select`, `Omit`

```go
db.Model(&user).Select("name").Updates(map[string]interface{}{"name": "hello", "age": 18, "actived": false})
//// UPDATE users SET name='hello', updated_at='2013-11-17 21:34:10' WHERE id=111;

db.Model(&user).Omit("name").Updates(map[string]interface{}{"name": "hello", "age": 18, "actived": false})
//// UPDATE users SET age=18, actived=false, updated_at='2013-11-17 21:34:10' WHERE id=111;
```

### Update Changed Fields Without Callbacks

Updating operations above will invoke `BeforeUpdate`, `AfterUpdate`, Update UpdatedAt timestamp, Save Associations callbacks, if you don't call them, you could use `UpdateColumn`, `UpdateColumns`

```go
// Update single attribute, similar with `Update`
db.Model(&user).UpdateColumn("name", "hello")
//// UPDATE users SET name='hello' WHERE id = 111;

// Update multiple attributes, similar with `Updates`
db.Model(&user).UpdateColumns(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18 WHERE id = 111;
```

### Batch Updates

Callbacks won't run when do batch updates

```go
db.Table("users").Where("id IN (?)", []int{10, 11}).Updates(map[string]interface{}{"name": "hello", "age": 18})
//// UPDATE users SET name='hello', age=18 WHERE id IN (10, 11);

// Update with struct only works with none zero values, or use map[string]interface{}
db.Model(User{}).Updates(User{Name: "hello", Age: 18})
//// UPDATE users SET name='hello', age=18;

// Get updated records count with `RowsAffected`
db.Model(User{}).Updates(User{Name: "hello", Age: 18}).RowsAffected
```

### Update with SQL Expression

```go
DB.Model(&product).Update("price", gorm.Expr("price * ? + ?", 2, 100))
//// UPDATE "products" SET "price" = price * '2' + '100', "updated_at" = '2013-11-17 21:34:10' WHERE "id" = '2';

DB.Model(&product).Updates(map[string]interface{}{"price": gorm.Expr("price * ? + ?", 2, 100)})
//// UPDATE "products" SET "price" = price * '2' + '100', "updated_at" = '2013-11-17 21:34:10' WHERE "id" = '2';

DB.Model(&product).UpdateColumn("quantity", gorm.Expr("quantity - ?", 1))
//// UPDATE "products" SET "quantity" = quantity - 1 WHERE "id" = '2';

DB.Model(&product).Where("quantity > 1").UpdateColumn("quantity", gorm.Expr("quantity - ?", 1))
//// UPDATE "products" SET "quantity" = quantity - 1 WHERE "id" = '2' AND quantity > 1;
```

### Change Updating Values In Callbacks

If you want to change updating values in callbacks using `BeforeUpdate`, `BeforeSave`, you could use `scope.SetColumn`, for example:

```go
func (user *User) BeforeSave(scope *gorm.Scope) (err error) {
  if pw, err := bcrypt.GenerateFromPassword(user.Password, 0); err == nil {
    scope.SetColumn("EncryptedPassword", pw)
  }
}
```

### Extra Updating option

```go
// Add extra SQL option for updating SQL
db.Model(&user).Set("gorm:update_option", "OPTION (OPTIMIZE FOR UNKNOWN)").Update("name, "hello")
//// UPDATE users SET name='hello', updated_at = '2013-11-17 21:34:10' WHERE id=111 OPTION (OPTIMIZE FOR UNKNOWN);
```
