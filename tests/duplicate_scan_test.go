package tests_test

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"testing"
)

type ProductD struct {
	*gorm.Model
	ID    int64  `json:"id" gorm:"id"`
	Code  string `json:"code" gorm:"type:varchar(20);uniqueIndex:idx_code"` // 指定类型为varchar(20)，创建唯一索引
	Price int64  `json:"price" gorm:"price"`
}

// TableName 表名称
func (*ProductD) TableName() string {
	return "products"
}

func (*ProductD) Name() string {
	return "products_d"
}

func TestDuplicateScan(t *testing.T) {
	DB.AutoMigrate(&ProductD{})
	DB.Create(&ProductD{Code: "D41", Price: 100})
	DB.Create(&ProductD{Code: "D42", Price: 100})
	// Migrate the schema
	dst := &ProductD{
		Code:  "D42",
		Price: 0,
	}
	tx := DB.Begin()
	tx = tx.Table(dst.Name()).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "code"}}, DoUpdates: clause.Assignments(map[string]interface{}{"price": gorm.Expr("price + ?", dst.Price)})}).Create(dst)
	p := new(Product)
	tx.Debug().Scan(&p)
	fmt.Printf("tx scan ||%+v\n", p)
	tx.Commit()
}
