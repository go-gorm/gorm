package conditional

import (
	"fmt"
	"gorm.io/gorm"
	"reflect"
	"regexp"
	"strings"
	"unicode"
	"unsafe"
)

const GeneralSumKey = "#sum"

var isInjectionReg = regexp.MustCompile("[^a-zA-Z_]+")

type BaseCondition struct {
	Page     *int    `json:"page" form:"page"`         // zh: 页码
	Pagesize *int    `json:"pagesize" form:"pagesize"` // zh: 每页大小
	OrderKey *string `json:"orderKey" form:"orderKey"` // zh: 排序字段名
	Desc     *bool   `json:"desc" form:"desc"`         // zh: 是否降序
}

type GeneralResult struct {
	Total int64                    `json:"total"`
	List  []map[string]interface{} `json:"list"`
	Sum   map[string]interface{}   `json:"sum"`
}

type wrapDB struct {
	Db                 *gorm.DB
	maxPagesize        int
	includeEmptyString int
	Page               *int    `json:"page" form:"page"`         // zh: 页码
	Pagesize           *int    `json:"pagesize" form:"pagesize"` // zh: 每页大小
	OrderKey           *string `json:"orderKey" form:"orderKey"` // zh: 排序字段名
	Desc               *bool   `json:"desc" form:"desc"`         // zh: 是否降序
}

func camelCaseToUnderscore(s string) string {
	var output []rune
	output = append(output, unicode.ToLower(rune(s[0])))
	for i := 1; i < len(s); i++ {
		if unicode.IsUpper(rune(s[i])) {
			output = append(output, '_')
		}
		output = append(output, unicode.ToLower(rune(s[i])))
	}
	return string(output)
}

func underscoreToUpperCamelCase(s string) string {
	var output []rune
	for i, f := 0, false; i < len(s); i++ {
		if s[i] == '_' {
			f = true
			continue
		}
		if f {
			f = false
			output = append(output, unicode.ToUpper(rune(s[i])))
		} else {
			output = append(output, rune(s[i]))
		}
	}
	return string(output)
}

func (w *wrapDB) doOrder() *wrapDB {
	if w.OrderKey != nil {
		if orderKey := *w.OrderKey; orderKey != "" && !isInjectionReg.MatchString(orderKey) {
			if w.Desc != nil && *w.Desc {
				orderKey += " desc"
			}
			w.Db.Order(orderKey)
		}
	}
	return w
}

func (w *wrapDB) doPage() *wrapDB {
	if w.Pagesize == nil || *w.Pagesize > w.maxPagesize {
		w.Pagesize = &w.maxPagesize
	}
	if *w.Pagesize > 0 {
		w.Db.Limit(*w.Pagesize)
	}
	if w.Page != nil {
		w.Db.Offset(*w.Pagesize * (*w.Page - 1))
	}
	return w
}

func (w *wrapDB) doWhere(key string, val interface{}) *wrapDB {
	if len(key) == 0 || strings.HasPrefix(key, "#") {
		return w
	}
	if w.includeEmptyString < 1 {
		if ref := reflect.ValueOf(val); ref.Kind() == reflect.String && ref.String() == "" {
			return w
		}
	}
	db := w.Db
	if key = camelCaseToUnderscore(key); len(key) > 3 {
		pre := key[:3]
		switch pre {
		case "neq":
			if key[3] == '_' {
				db.Where(fmt.Sprintf("`%s` <> ?", key[4:]), val)
			}
		case "gt_":
			db.Where(fmt.Sprintf("`%s` >= ?", key[3:]), val)
		case "lt_":
			db.Where(fmt.Sprintf("`%s` <= ?", key[3:]), val)
		case "in_":
			db.Where(fmt.Sprintf("`%s` in ?", key[3:]), val)
		case "nin":
			if key[3] == '_' {
				db.Where(fmt.Sprintf("`%s` not in ?", key[4:]), val)
			}
		case "lik":
			if strings.HasPrefix(key, "like_") {
				db.Where(fmt.Sprintf("`%s` like ?", key[5:]), val)
			}
		case "nli":
			if strings.HasPrefix(key, "nlike_") {
				db.Where(fmt.Sprintf("`%s` not like ?", key[6:]), val)
			}
		case "pag":
			if key == "page" {
				var page int
				ref := reflect.ValueOf(val)
				if ref.CanFloat() {
					page = int(ref.Float())
				} else if ref.CanInt() {
					page = int(ref.Int())
				} else {
					page = int(ref.Uint())
				}
				w.Page = &page
			} else if key == "pagesize" || key == "page_size" {
				var pagesize int
				ref := reflect.ValueOf(val)
				if ref.CanFloat() {
					pagesize = int(ref.Float())
				} else if ref.CanInt() {
					pagesize = int(ref.Int())
				} else {
					pagesize = int(ref.Uint())
				}
				w.Pagesize = &pagesize
			}
		case "ord":
			if key == "order_key" {
				v := camelCaseToUnderscore(val.(string))
				if strings.HasPrefix(v, "desc_") {
					n, b := v[5:], true
					w.OrderKey, w.Desc = &n, &b
				} else {
					var n string
					if strings.HasPrefix(v, "asc_") {
						n = v[4:]
					} else {
						n = v
					}
					w.OrderKey = &n
				}
			}
		case "eq_":
			db.Where(fmt.Sprintf("`%s` = ?", key[3:]), val)
		default:
			db.Where(fmt.Sprintf("`%s` = ?", key), val)
		}
	} else {
		db.Where(fmt.Sprintf("`%s` = ?", key), val)
	}
	return w
}

func (w *wrapDB) doDeepWhere(k string, v reflect.Value) *wrapDB {
	kind := v.Kind()
	switch kind {
	case reflect.Pointer, reflect.UnsafePointer:
		if !v.IsNil() {
			w.doDeepWhere(k, v.Elem())
		}
	case reflect.Struct:
		t, n := v.Type(), v.NumField()
		for i := 0; i < n; i++ {
			ki, vi := t.Field(i).Name, v.Field(i)
			kind = vi.Kind()
			switch kind {
			case reflect.Pointer, reflect.UnsafePointer:
				w.doDeepWhere(ki, vi.Elem())
			case reflect.Struct, reflect.Map:
				w.doDeepWhere("", vi)
			default:
				w.doWhere(ki, vi.Interface())
			}
		}
	case reflect.Map:
		if keys := v.MapKeys(); len(keys) > 0 && keys[0].Kind() == reflect.String {
			for _, key := range keys {
				w.doWhere(key.String(), v.MapIndex(key).Interface())
			}
		}
	default:
		w.doWhere(k, v.Interface())
	}
	return w
}

func QueryGeneralConditional(db *gorm.DB, search map[string]interface{}, maxPagesize, includeEmptyString int) (gr GeneralResult, err error) {
	gr = GeneralResult{List: make([]map[string]interface{}, 0), Sum: make(map[string]interface{}), Total: 0}
	wdb := &wrapDB{Db: db, maxPagesize: maxPagesize, includeEmptyString: includeEmptyString}
	for key, val := range search {
		wdb.doWhere(key, val)
	}
	if err = wdb.Db.Count(&gr.Total).Error; err != nil {
		return
	}
	if gr.Total > 0 {
		wdb.doOrder().doPage()
		if err = wdb.Db.Scan(&gr.List).Error; err != nil {
			return
		}
		// [Optional] Underscore field name to UpperCamelCase
		if len(gr.List) > 0 {
			list := make([]map[string]interface{}, 0)
			for _, m := range gr.List {
				nm := make(map[string]interface{})
				for k, v := range m {
					nm[underscoreToUpperCamelCase(k)] = v
				}
				list = append(list, nm)
			}
			gr.List = list
		}
		// page-1 can do sum only
		if wdb.Page != nil && *wdb.Page == 1 {
			var sumFields []string
			if sumKeys := search[GeneralSumKey]; sumKeys != nil {
				for _, sumKey := range sumKeys.([]string) {
					if isInjectionReg.MatchString(sumKey) {
						err = fmt.Errorf("Ilegal sumKey: %s ", sumKey)
						return
					}
					sumFields = append(sumFields, camelCaseToUnderscore(sumKey))
				}
			}
			if len(sumFields) > 0 {
				var sb strings.Builder
				for _, field := range sumFields {
					sb.Write([]byte(fmt.Sprintf("sum(`%s`) as `%s`, ", field, field)))
				}
				if sb.Len() > 16 {
					if err = db.Select(sb.String()[:sb.Len()-2]).Scan(gr.Sum).Error; err != nil {
						return
					}
				}
			}
		}
	}
	return
}

func QueryStructConditional(db *gorm.DB, search interface{}, list interface{}, sum interface{}, total *int64, maxPagesize, includeEmptyString int) (err error) {
	wdb := &wrapDB{Db: db, maxPagesize: maxPagesize, includeEmptyString: includeEmptyString}
	if search != nil {
		wdb.doDeepWhere("", reflect.ValueOf(search))
	}
	if total != nil {
		if err = wdb.Db.Count(total).Error; err != nil {
			return
		}
	}
	if *total > 0 {
		wdb.doOrder().doPage()
		if list != nil {
			if err = wdb.Db.Scan(list).Error; err != nil {
				return
			}
		}
		// page-1 can do sum only
		if wdb.Page != nil && *wdb.Page == 1 && sum != nil {
			sv := reflect.ValueOf(sum)
			sk := sv.Kind()
			switch sk {
			case reflect.Pointer, reflect.UnsafePointer:
				sv = reflect.ValueOf(sv.Elem().Interface())
				sk = sv.Kind()
				if sk == reflect.Struct {
					if t, n := sv.Type(), sv.NumField(); n > 0 {
						var sb strings.Builder
						for i := 0; i < n; i++ {
							ns := camelCaseToUnderscore(t.Field(i).Name)
							sb.Write([]byte(fmt.Sprintf("sum(`%s`) as `%s`, ", ns, ns)))
						}
						if sb.Len() > 0 {
							db.Select(sb.String()[:sb.Len()-2]).Scan(sum)
						}
					}
				}
			default:
				return fmt.Errorf("[SUM] Unknow: %s , only pointer allowed", sum)
			}
		}
	} else {
		if list != nil {
			// set empty array
			s := (*reflect.SliceHeader)(reflect.ValueOf(list).UnsafePointer())
			if s.Data == 0 {
				e := make([]interface{}, 0)
				s.Data = (uintptr)(unsafe.Pointer(&e))
			}
		}
	}
	return
}
