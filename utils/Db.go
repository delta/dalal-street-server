package utils

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

var dbConfig *Config

// DbOpen returns a database connection object by opening one based
// on the configuration
func DbOpen() (*gorm.DB, error) {
	user := dbConfig.DbUser
	pwd := dbConfig.DbPassword
	host := dbConfig.DbHost
	dbname := dbConfig.DbName

	connstr := fmt.Sprintf("%s:%s@%s/%s?charset=utf8&parseTime=true", user, pwd, host, dbname)
	return gorm.Open("mysql", connstr)
}

func initDbHelper(config *Config) {
	fmt.Printf("initDbHelper called!")
	dbConfig = config
}
