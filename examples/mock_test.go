package examples
import (
	"testing"
	"github.com/jinzhu/gorm/mocks"
	"github.com/stretchr/testify/mock"
)

var databaseMock *mocks.Database
var expected []interface{}

func setUp(){
	databaseMock = new(mocks.Database)

	databaseMock.On("Where", mock.Anything, mock.Anything).Return(databaseMock)
	databaseMock.On("Find", mock.Anything, expected).Return(databaseMock)
	databaseMock.On("GetError").Return(nil)
}

func TestFindAllWithoutFilter(t *testing.T){
	setUp()

	repo := ProductRepositoryImpl{DB: databaseMock}

	products, err := repo.FindAll(-1, -1)

	databaseMock.AssertCalled(t, "Find", &products, expected)

	if err != nil {
		t.Error("No error expected")
	}
}

func TestFindAll(t *testing.T) {
	setUp()

	repo := ProductRepositoryImpl{DB: databaseMock}

	repo.FindAll(1, 2)

	databaseMock.AssertCalled(t, "Where", "storeId = ?", []interface{}{1})
	databaseMock.AssertCalled(t, "Where", "categoryId = ?", []interface{}{2})
}
