# Logger

Gorm has built-in logger support

```go
// Enable Logger
db.LogMode(true)

// Diable Logger
db.LogMode(false)

// Debug a single operation
db.Debug().Where("name = ?", "jinzhu").First(&User{})
```

![logger](https://raw.github.com/jinzhu/gorm/master/doc/logger.png)

### Customize Logger

```go
// Refer gorm's default logger for how to: https://github.com/jinzhu/gorm/blob/master/logger.go#files
db.SetLogger(gorm.Logger{revel.TRACE})
db.SetLogger(log.New(os.Stdout, "\r\n", 0))
```
