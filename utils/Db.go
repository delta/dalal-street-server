package utils

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

// DbOpen() returns a database connection object by opening one based
// on the configuration
func DbOpen() (*gorm.DB, error) {
	user := Configuration.DbUser
	pwd := Configuration.DbPassword
	host := Configuration.DbHost
	dbname := Configuration.DbName

	connstr := fmt.Sprintf("%s:%s@%s/%s?charset=utf8&parseTime=true", user, pwd, host, dbname)
	return gorm.Open("mysql", connstr)
}
