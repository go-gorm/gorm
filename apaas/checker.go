package apaas

import (
	"encoding/json"
	"fmt"
)

type Checker interface {
	Check(string) error
}

func ExtraCheck(rule map[string]*ExtraFieldMeta, extra string) error {
	var data map[string]any
	err := json.Unmarshal([]byte(extra), &data)
	if err != nil {
		return err
	}
	for k, v := range data {
		r, ok := rule[k]
		if !ok || r == nil {
			continue
		}
		err = FieldCheck(r, v)
		if err != nil {
			return GenError(err.Error())
		}
	}
	return nil
}

func FieldCheck(rule *ExtraFieldMeta, value any) error {
	if rule == nil || value == nil {
		return nil
	}
	var err error
	switch val := value.(type) {
	case map[string]any:
		if len(rule.ObjectMeta) == 0 {
			return nil
		}
		for k, v := range val {
			r, ok := rule.ObjectMeta[k]
			if !ok || r == nil {
				continue
			}
			err = FieldCheck(r, v)
			if err != nil {
				return fmt.Errorf("rule(key=%s, type=%s), value(type=map[%s], suberror=%s)",
					rule.Key, FieldMapString[rule.Type], k, err.Error())
			}
		}
	case []any:
		if rule.Type != FieldArray {
			return fmt.Errorf("rule(key=%s, type=%s) dismatch value(type=array, value=%#v)",
				rule.Key, FieldMapString[rule.Type], val)
		}
		if len(rule.ArrayMeta) != 0 {
			for i, v := range val {
				err = FieldCheck(rule.ArrayMeta[i], v)
				if err != nil {
					return fmt.Errorf("rule(key=%s, type=%s), value(type array, [%d] element suberror=%s)",
						rule.Key, FieldMapString[rule.Type], i, err.Error())
				}
			}
		}
	case string:
		if rule.Type != FieldString {
			return fmt.Errorf("rule(key=%s, type=%s) dismatch value(type=string, value=%#v)",
				rule.Key, FieldMapString[rule.Type], val)
		}
	case float64:
		if rule.Type != FieldInt && rule.Type != FieldFloat64 {
			return fmt.Errorf("rule(key=%s, type=%s) dismatch value(type=float/int, value=%#v)",
				rule.Key, FieldMapString[rule.Type], val)
		}
		if rule.Type == FieldInt && float64(int(val)) != val {
			return fmt.Errorf("rule(key=%s, type=%s) dismatch value(type=float, value=%#v)",
				rule.Key, FieldMapString[rule.Type], val)
		}
	case bool:
		if rule.Type != FieldBool {
			return fmt.Errorf("rule(key=%s, type=%s) dismatch value(type=bool, value=%#v)",
				rule.Key, FieldMapString[rule.Type])
		}
	case nil:
		if rule.Type != FieldArray || rule.Type != FieldObject {
			return fmt.Errorf("rule(key=%s, type=%s) dismatch value(type nil)",
				rule.Key, FieldMapString[rule.Type])
		}
	}
	return nil
}
