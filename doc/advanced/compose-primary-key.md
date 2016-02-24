# Composite Primary Key

```go
type Product struct {
	ID           string `gorm:"primary_key"`
	LanguageCode string `gorm:"primary_key"`
}
```
