# Error Handling

```go
query := db.Where("name = ?", "jinzhu").First(&user)
query := db.First(&user).Limit(10).Find(&users)
// query.Error will return the last happened error

// So you could do error handing in your application like this:
if err := db.Where("name = ?", "jinzhu").First(&user).Error; err != nil {
	// error handling...
}

// RecordNotFound
// If no record found when you query data, gorm will return RecordNotFound error, you could check it like this:
db.Where("name = ?", "hello world").First(&User{}).Error == gorm.RecordNotFound
// Or use the shortcut method
db.Where("name = ?", "hello world").First(&user).RecordNotFound()

if db.Model(&user).Related(&credit_card).RecordNotFound() {
	// no credit card found error handling
}
```
