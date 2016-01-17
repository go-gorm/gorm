package examples
import (
	"github.com/jinzhu/gorm"
)

type Product struct {
	gorm.Model
	Name       string
	Price      float64
	StoreId    int64
	CategoryId int64
}

type ProductRepository interface {
	FindAll(storeId int64, categoryId int64) ([]Product, error)
}

type ProductRepositoryImpl struct {
	DB gorm.Database
}

func (r *ProductRepositoryImpl) FindAll(storeId int64, categoryId int64) ([]Product, error) {
	qb := r.DB
	if storeId > 0 {
		qb = qb.Where("storeId = ?", storeId)
	}

	if categoryId > 0 {
		qb = qb.Where("categoryId = ?", categoryId)
	}

	var products []Product
	err := qb.Find(&products).GetError()

	return products, err
}
