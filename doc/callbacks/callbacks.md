# Callbacks

Callbacks are methods defined on the pointer of struct.
If any callback returns an error, gorm will stop future operations and rollback all changes.

Here is the list of all available callbacks:
(listed in the same order in which they will get called during the respective operations)

### Creating An Object

```go
BeforeSave
BeforeCreate
// save before associations
// save self
// save after associations
AfterCreate
AfterSave
```
### Updating An Object

```go
BeforeSave
BeforeUpdate
// save before associations
// save self
// save after associations
AfterUpdate
AfterSave
```

### Destroying An Object

```go
BeforeDelete
// delete self
AfterDelete
```

### After Find

```go
// load data from database
AfterFind
```

### Example

```go
func (u *User) BeforeUpdate() (err error) {
	if u.readonly() {
		err = errors.New("read only user")
	}
	return
}

// Rollback the insertion if user's id greater than 1000
func (u *User) AfterCreate() (err error) {
	if (u.Id > 1000) {
		err = errors.New("user id is already greater than 1000")
	}
	return
}
```

Save/delete operations in gorm are running in a transaction.
Changes made in that transaction are not visible unless it is commited.
So if you want to use those changes in your callbacks, you need to run your SQL in the same transaction.
For this Gorm supports passing transactions to callbacks like this:

```go
func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	tx.Model(u).Update("role", "admin")
	return
}
```
