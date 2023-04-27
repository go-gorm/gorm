package conditional

import (
	"log"
	"testing"
)

type User struct {
	Id        uint   `json:"id"`
	Name      string `json:"name"`
	Level     uint   `json:"level"`
	Status    uint   `json:"status"`
	CreatedAt uint   `json:"createdAt"`
	UpdatedAt uint   `json:"updatedAt"`
}

func TestQueryStructConditionalNeq(t *testing.T) {
	search := struct {
		NeqId uint
	}{NeqId: 1}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalEq(t *testing.T) {
	//search := struct {
	//	Id uint
	//}{Id: 1}
	search := struct {
		EqId uint
	}{EqId: 1}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalLt(t *testing.T) {
	search := struct {
		LtId uint
	}{LtId: 1}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalGt(t *testing.T) {
	search := struct {
		GtId uint
	}{GtId: 1}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalIn(t *testing.T) {
	search := struct {
		InId []uint
	}{InId: []uint{1, 2}}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalNin(t *testing.T) {
	search := struct {
		NinId []uint
	}{NinId: []uint{1, 2}}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalLike(t *testing.T) {
	search := struct {
		LikeName string
	}{LikeName: "f%"}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalNlike(t *testing.T) {
	search := struct {
		NlikeName string
	}{NlikeName: "f%"}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalPage(t *testing.T) {
	search := struct {
		Page     int
		Pagesize int
	}{Page: 2, Pagesize: 2}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalOrder(t *testing.T) {
	search := struct {
		OrderKey string
	}{OrderKey: "descId"}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalPage1Sum(t *testing.T) {
	search := struct {
		Page int
	}{Page: 1}
	sum := new(struct {
		Level uint
	})
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, sum, total, 10, 1)
	if err != nil {
		log.Println(err)
	}
	log.Println(sum, list, *total)
}

func TestQueryStructConditionalNotAllowEmptyString(t *testing.T) {
	search := struct {
		Name string
	}{Name: ""}
	//search := struct {
	//	Name string
	//}{Name: "foo"}
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), search, list, nil, total, 10, 0)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}

func TestQueryStructConditionalMaxCount(t *testing.T) {
	list, total := new([]User), new(int64)
	err := QueryStructConditional(initDB(), nil, list, nil, total, 2, 0)
	if err != nil {
		log.Println(err)
	}
	log.Println(list, *total)
}
