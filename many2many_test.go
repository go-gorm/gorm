package gorm_test

import "testing"

func TestQueryManyToManyWithRelated(t *testing.T) {
	db.Model(&User{}).Related(&[]Language{}, "Languages")
	// SELECT `languages`.* FROM `languages` INNER JOIN `user_languages` ON `languages`.`id` = `user_languages`.`language_id` WHERE `user_languages`.`user_id` = 111
	// db.Model(&User{}).Many2Many("Languages").Find(&[]Language{})
	// db.Model(&User{}).Many2Many("Languages").Add(&Language{})
	// db.Model(&User{}).Many2Many("Languages").Remove(&Language{})
	// db.Model(&User{}).Many2Many("Languages").Replace(&[]Language{})
}
