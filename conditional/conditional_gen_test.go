package conditional

import (
	"encoding/json"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"testing"
)

func initDB() *gorm.DB {
	db, err := gorm.Open(mysql.Open("root:root@tcp(127.0.0.1:3306)/gorm_test?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{Logger: logger.Default.LogMode(logger.Info)})
	if err != nil {
		panic("failed to connect database")
	}
	// `id`         INT unsigned NOT NULL AUTO_INCREMENT COMMENT 'user ID',
	// `name`       VARCHAR(64)  NOT NULL COMMENT '钱包地址',
	// `level`      INT unsigned NOT NULL COMMENT '用户等级',
	// `status`     int unsigned NOT NULL DEFAULT '0' COMMENT '结算状态 0: 正常  20禁用',
	// `created_at` bigint       NOT NULL COMMENT '创建时间 毫秒',
	// `updated_at` bigint       NOT NULL COMMENT '更新时间 毫秒',
	db = db.Table("user")
	return db
}

func TestQueryGeneralConditionalNeq(t *testing.T) {
	search := make(map[string]interface{})
	search["neqId"] = uint(1)
	search["neq_id"] = uint(1)
	gr, err := QueryGeneralConditional(initDB(), search, 10, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalEq(t *testing.T) {
	search := make(map[string]interface{})
	search["id"] = uint(1)
	search["eq_id"] = uint(1)
	search["eqId"] = uint(1)
	gr, err := QueryGeneralConditional(initDB(), search, 10, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalLt(t *testing.T) {
	search := make(map[string]interface{})
	search["ltId"] = uint(2)
	gr, err := QueryGeneralConditional(initDB(), search, 10, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalGt(t *testing.T) {
	search := make(map[string]interface{})
	search["gtId"] = uint(2)
	gr, err := QueryGeneralConditional(initDB(), search, 10, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalIn(t *testing.T) {
	search := make(map[string]interface{})
	in := make([]uint, 0)
	in = append(in, 1)
	in = append(in, 2)
	in = append(in, 3)
	search["inId"] = in
	gr, err := QueryGeneralConditional(initDB(), search, 10, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalNin(t *testing.T) {
	search := make(map[string]interface{})
	in := make([]uint, 0)
	in = append(in, 1)
	in = append(in, 2)
	in = append(in, 3)
	search["ninId"] = in
	gr, err := QueryGeneralConditional(initDB(), search, 10, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalLike(t *testing.T) {
	search := make(map[string]interface{})
	search["likeName"] = "_oo"
	gr, err := QueryGeneralConditional(initDB(), search, 10, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalNlike(t *testing.T) {
	search := make(map[string]interface{})
	search["likeName"] = "f%"
	gr, err := QueryGeneralConditional(initDB(), search, 10, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalPage(t *testing.T) {
	search := make(map[string]interface{})
	search["page"] = 2
	search["pagesize"] = 2
	gr, err := QueryGeneralConditional(initDB(), search, 10, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalOrder(t *testing.T) {
	search := make(map[string]interface{})
	//search["orderKey"] = "descId"
	search["orderKey"] = "Id"
	search["orderKey"] = "ascId"
	//search["orderKey"] = "descId or 1"
	gr, err := QueryGeneralConditional(initDB(), search, 2, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalPage1Sum(t *testing.T) {
	search := make(map[string]interface{})
	search["page"] = 1
	search[GeneralSumKey] = []string{"level"}
	gr, err := QueryGeneralConditional(initDB(), search, 2, 1)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalNotAllowEmptyString(t *testing.T) {
	search := make(map[string]interface{})
	search["likeName"] = ""
	gr, err := QueryGeneralConditional(initDB(), search, 2, 0)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))
	search["likeName"] = "f%"
	gr, err = QueryGeneralConditional(initDB(), search, 2, 0)
	if err != nil {
		log.Println(err)
	}
	marshal, _ = json.Marshal(gr)
	log.Println(string(marshal))
}

func TestQueryGeneralConditionalMaxCount(t *testing.T) {
	// [unsafe] Unrestricted mode : <=0
	search := make(map[string]interface{})
	gr, err := QueryGeneralConditional(initDB(), search, -1, 0)
	if err != nil {
		log.Println(err)
	}
	marshal, _ := json.Marshal(gr)
	log.Println(string(marshal))

	// [safe]  > 0
	search = make(map[string]interface{})
	gr, err = QueryGeneralConditional(initDB(), search, 1, 0)
	if err != nil {
		log.Println(err)
	}
	marshal, _ = json.Marshal(gr)
	log.Println(string(marshal))
}
