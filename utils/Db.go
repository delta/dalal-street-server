package utils

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var dbConfig *Config
var db *gorm.DB

// GetDB returns a database connection object by opening one based
// on the configuration
func GetDB() *gorm.DB {
	return db
}

func CloseDB() error {
	return db.Close()
}

func initDbHelper(config *Config) {
	fmt.Printf("initDbHelper called!")
	dbConfig = config

	user := dbConfig.DbUser
	pwd := dbConfig.DbPassword
	host := dbConfig.DbHost
	dbname := dbConfig.DbName

	connstr := fmt.Sprintf("%s:%s@%s/%s?charset=utf8&parseTime=true", user, pwd, host, dbname)

	var err error
	db, err = gorm.Open("mysql", connstr)
	if err != nil {
		panic(fmt.Errorf("Error opening DB. Got error: %+v", err))
	}
}
