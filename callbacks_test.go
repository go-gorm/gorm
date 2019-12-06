package gorm_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jinzhu/gorm"
)

func (s *Product) BeforeCreate() (err error) {
	if s.Code == "Invalid" {
		err = errors.New("invalid product")
	}
	s.BeforeCreateCallTimes = s.BeforeCreateCallTimes + 1
	return
}

func (s *Product) BeforeUpdate() (err error) {
	if s.Code == "dont_update" {
		err = errors.New("can't update")
	}
	s.BeforeUpdateCallTimes = s.BeforeUpdateCallTimes + 1
	return
}

func (s *Product) BeforeSave() (err error) {
	if s.Code == "dont_save" {
		err = errors.New("can't save")
	}
	s.BeforeSaveCallTimes = s.BeforeSaveCallTimes + 1
	return
}

func (s *Product) AfterFind() {
	s.AfterFindCallTimes = s.AfterFindCallTimes + 1
}

func (s *Product) AfterCreate(tx *gorm.DB) {
	tx.Model(s).UpdateColumn(Product{AfterCreateCallTimes: s.AfterCreateCallTimes + 1})
}

func (s *Product) AfterUpdate() {
	s.AfterUpdateCallTimes = s.AfterUpdateCallTimes + 1
}

func (s *Product) AfterSave() (err error) {
	if s.Code == "after_save_error" {
		err = errors.New("can't save")
	}
	s.AfterSaveCallTimes = s.AfterSaveCallTimes + 1
	return
}

func (s *Product) BeforeDelete() (err error) {
	if s.Code == "dont_delete" {
		err = errors.New("can't delete")
	}
	s.BeforeDeleteCallTimes = s.BeforeDeleteCallTimes + 1
	return
}

func (s *Product) AfterDelete() (err error) {
	if s.Code == "after_delete_error" {
		err = errors.New("can't delete")
	}
	s.AfterDeleteCallTimes = s.AfterDeleteCallTimes + 1
	return
}

func (s *Product) GetCallTimes() []int64 {
	return []int64{s.BeforeCreateCallTimes, s.BeforeSaveCallTimes, s.BeforeUpdateCallTimes, s.AfterCreateCallTimes, s.AfterSaveCallTimes, s.AfterUpdateCallTimes, s.BeforeDeleteCallTimes, s.AfterDeleteCallTimes, s.AfterFindCallTimes}
}

func TestRunCallbacks(t *testing.T) {
	p := Product{Code: "unique_code", Price: 100}
	DB.Save(&p)

	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 1, 0, 1, 1, 0, 0, 0, 0}) {
		t.Errorf("Callbacks should be invoked successfully, %v", p.GetCallTimes())
	}

	DB.Where("Code = ?", "unique_code").First(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 1, 0, 1, 0, 0, 0, 0, 1}) {
		t.Errorf("After callbacks values are not saved, %v", p.GetCallTimes())
	}

	p.Price = 200
	DB.Save(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 2, 1, 1, 1, 1, 0, 0, 1}) {
		t.Errorf("After update callbacks should be invoked successfully, %v", p.GetCallTimes())
	}

	var products []Product
	DB.Find(&products, "code = ?", "unique_code")
	if products[0].AfterFindCallTimes != 2 {
		t.Errorf("AfterFind callbacks should work with slice")
	}

	DB.Where("Code = ?", "unique_code").First(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 2, 1, 1, 0, 0, 0, 0, 2}) {
		t.Errorf("After update callbacks values are not saved, %v", p.GetCallTimes())
	}

	DB.Delete(&p)
	if !reflect.DeepEqual(p.GetCallTimes(), []int64{1, 2, 1, 1, 0, 0, 1, 1, 2}) {
		t.Errorf("After delete callbacks should be invoked successfully, %v", p.GetCallTimes())
	}

	if DB.Where("Code = ?", "unique_code").First(&p).Error == nil {
		t.Errorf("Can't find a deleted record")
	}
}

func TestCallbacksWithErrors(t *testing.T) {
	p := Product{Code: "Invalid", Price: 100}
	if DB.Save(&p).Error == nil {
		t.Errorf("An error from before create callbacks happened when create with invalid value")
	}

	if DB.Where("code = ?", "Invalid").First(&Product{}).Error == nil {
		t.Errorf("Should not save record that have errors")
	}

	if DB.Save(&Product{Code: "dont_save", Price: 100}).Error == nil {
		t.Errorf("An error from after create callbacks happened when create with invalid value")
	}

	p2 := Product{Code: "update_callback", Price: 100}
	DB.Save(&p2)

	p2.Code = "dont_update"
	if DB.Save(&p2).Error == nil {
		t.Errorf("An error from before update callbacks happened when update with invalid value")
	}

	if DB.Where("code = ?", "update_callback").First(&Product{}).Error != nil {
		t.Errorf("Record Should not be updated due to errors happened in before update callback")
	}

	if DB.Where("code = ?", "dont_update").First(&Product{}).Error == nil {
		t.Errorf("Record Should not be updated due to errors happened in before update callback")
	}

	p2.Code = "dont_save"
	if DB.Save(&p2).Error == nil {
		t.Errorf("An error from before save callbacks happened when update with invalid value")
	}

	p3 := Product{Code: "dont_delete", Price: 100}
	DB.Save(&p3)
	if DB.Delete(&p3).Error == nil {
		t.Errorf("An error from before delete callbacks happened when delete")
	}

	if DB.Where("Code = ?", "dont_delete").First(&p3).Error != nil {
		t.Errorf("An error from before delete callbacks happened")
	}

	p4 := Product{Code: "after_save_error", Price: 100}
	DB.Save(&p4)
	if err := DB.First(&Product{}, "code = ?", "after_save_error").Error; err == nil {
		t.Errorf("Record should be reverted if get an error in after save callback")
	}

	p5 := Product{Code: "after_delete_error", Price: 100}
	DB.Save(&p5)
	if err := DB.First(&Product{}, "code = ?", "after_delete_error").Error; err != nil {
		t.Errorf("Record should be found")
	}

	DB.Delete(&p5)
	if err := DB.First(&Product{}, "code = ?", "after_delete_error").Error; err != nil {
		t.Errorf("Record shouldn't be deleted because of an error happened in after delete callback")
	}
}

func TestGetCallback(t *testing.T) {
	scope := DB.NewScope(nil)

	if DB.Callback().Create().Get("gorm:test_callback") != nil {
		t.Errorf("`gorm:test_callback` should be nil")
	}

	DB.Callback().Create().Register("gorm:test_callback", func(scope *gorm.Scope) { scope.Set("gorm:test_callback_value", 1) })
	callback := DB.Callback().Create().Get("gorm:test_callback")
	if callback == nil {
		t.Errorf("`gorm:test_callback` should be non-nil")
	}
	callback(scope)
	if v, ok := scope.Get("gorm:test_callback_value"); !ok || v != 1 {
		t.Errorf("`gorm:test_callback_value` should be `1, true` but `%v, %v`", v, ok)
	}

	DB.Callback().Create().Replace("gorm:test_callback", func(scope *gorm.Scope) { scope.Set("gorm:test_callback_value", 2) })
	callback = DB.Callback().Create().Get("gorm:test_callback")
	if callback == nil {
		t.Errorf("`gorm:test_callback` should be non-nil")
	}
	callback(scope)
	if v, ok := scope.Get("gorm:test_callback_value"); !ok || v != 2 {
		t.Errorf("`gorm:test_callback_value` should be `2, true` but `%v, %v`", v, ok)
	}

	DB.Callback().Create().Remove("gorm:test_callback")
	if DB.Callback().Create().Get("gorm:test_callback") != nil {
		t.Errorf("`gorm:test_callback` should be nil")
	}

	DB.Callback().Create().Register("gorm:test_callback", func(scope *gorm.Scope) { scope.Set("gorm:test_callback_value", 3) })
	callback = DB.Callback().Create().Get("gorm:test_callback")
	if callback == nil {
		t.Errorf("`gorm:test_callback` should be non-nil")
	}
	callback(scope)
	if v, ok := scope.Get("gorm:test_callback_value"); !ok || v != 3 {
		t.Errorf("`gorm:test_callback_value` should be `3, true` but `%v, %v`", v, ok)
	}
}

func TestUseDefaultCallback(t *testing.T) {
	createCallbackName := "gorm:test_use_default_callback_for_create"
	gorm.DefaultCallback.Create().Register(createCallbackName, func(*gorm.Scope) {
		// nop
	})
	if gorm.DefaultCallback.Create().Get(createCallbackName) == nil {
		t.Errorf("`%s` expected non-nil, but got nil", createCallbackName)
	}
	gorm.DefaultCallback.Create().Remove(createCallbackName)
	if gorm.DefaultCallback.Create().Get(createCallbackName) != nil {
		t.Errorf("`%s` expected nil, but got non-nil", createCallbackName)
	}

	updateCallbackName := "gorm:test_use_default_callback_for_update"
	scopeValueName := "gorm:test_use_default_callback_for_update_value"
	gorm.DefaultCallback.Update().Register(updateCallbackName, func(scope *gorm.Scope) {
		scope.Set(scopeValueName, 1)
	})
	gorm.DefaultCallback.Update().Replace(updateCallbackName, func(scope *gorm.Scope) {
		scope.Set(scopeValueName, 2)
	})

	scope := DB.NewScope(nil)
	callback := gorm.DefaultCallback.Update().Get(updateCallbackName)
	callback(scope)
	if v, ok := scope.Get(scopeValueName); !ok || v != 2 {
		t.Errorf("`%s` should be `2, true` but `%v, %v`", scopeValueName, v, ok)
	}
}
