# Delete

```go
// Delete an existing record
db.Delete(&email)
//// DELETE from emails where id=10;

// Add extra SQL option for deleting SQL
db.Set("gorm:delete_option", "OPTION (OPTIMIZE FOR UNKNOWN)").Delete(&email)
//// DELETE from emails where id=10 OPTION (OPTIMIZE FOR UNKNOWN);
```

### Batch Delete

```go
db.Where("email LIKE ?", "%jinzhu%").Delete(Email{})
//// DELETE from emails where email LIKE "%jinhu%";
```

### Soft Delete

If struct has `DeletedAt` field, it will get soft delete ability automatically!
Then it won't be deleted from database permanently when call `Delete`.

```go
db.Delete(&user)
//// UPDATE users SET deleted_at="2013-10-29 10:23" WHERE id = 111;

// Batch Delete
db.Where("age = ?", 20).Delete(&User{})
//// UPDATE users SET deleted_at="2013-10-29 10:23" WHERE age = 20;

// Soft deleted records will be ignored when query them
db.Where("age = 20").Find(&user)
//// SELECT * FROM users WHERE age = 20 AND deleted_at IS NULL;

// Find soft deleted records with Unscoped
db.Unscoped().Where("age = 20").Find(&users)
//// SELECT * FROM users WHERE age = 20;

// Delete record permanently with Unscoped
db.Unscoped().Delete(&order)
//// DELETE FROM orders WHERE id=10;
```
