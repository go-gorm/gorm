package migrations

import "fmt"

func MigrateAll() {
	fmt.Println("Running migrations...")
	UpUser()
	// Add other migrations here
	fmt.Println("Migrations completed!")
}

func RollbackAll() {
	fmt.Println("Rolling back migrations...")
	DownUser()
	// Add other rollbacks here
	fmt.Println("Rollback completed!")
}
