package Connection

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectMySQL(username, password, host, port, name string) (*gorm.DB, error) {
	// Format DSN (Data Source Name)
	dsn := username + ":" + password + "@tcp(" + host + ":" + port + ")/" + name + "?charset=utf8mb4&parseTime=True&loc=Local"
	fmt.Println("Connecting to MySQL with DSN:", dsn)
	// Membuka koneksi
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	DB = db
	return db, nil
}
