package tests

import (
	"strconv"
	"time"
)

type Config struct {
	Account   bool
	Pets      int
	Toys      int
	Company   bool
	Manager   bool
	Team      int
	Languages int
	Friends   int
}

func GetUser(name string, config Config) User {
	var (
		birthday = time.Now()
		user     = User{
			Name:     name,
			Age:      18,
			Birthday: &birthday,
		}
	)

	if config.Account {
		user.Account = Account{Number: name + "_account"}
	}

	for i := 0; i < config.Pets; i++ {
		user.Pets = append(user.Pets, &Pet{Name: name + "_pet_" + strconv.Itoa(i+1)})
	}

	for i := 0; i < config.Toys; i++ {
		user.Toys = append(user.Toys, Toy{Name: name + "_toy_" + strconv.Itoa(i+1)})
	}

	if config.Company {
		user.Company = Company{Name: "company-" + name}
	}

	if config.Manager {
		manager := GetUser(name+"_manager", Config{})
		user.Manager = &manager
	}

	for i := 0; i < config.Team; i++ {
		user.Team = append(user.Team, GetUser(name+"_team_"+strconv.Itoa(i+1), Config{}))
	}

	for i := 0; i < config.Languages; i++ {
		name := "Locale_" + strconv.Itoa(i+0)
		user.Languages = append(user.Languages, Language{Code: name, Name: name})
	}

	for i := 0; i < config.Friends; i++ {
		f := GetUser(name+"_friend_"+strconv.Itoa(i+1), Config{})
		user.Friends = append(user.Friends, &f)
	}

	return user
}
