// Package models handles everything between the database and the API.
// All business logic is written in this package, so the user of this
// package does not need to take care of race conditions in updating
// the data.
package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"

	"github.com/thakkarparth007/dalal-street-server/utils"
)

// DbOpen() returns a database connection object by opening one based
// on the configuration
func DbOpen() (*gorm.DB, error) {
	user := utils.Configuration.DbUser
	pwd := utils.Configuration.DbPassword
	host := utils.Configuration.DbHost
	dbname := utils.Configuration.DbName

	connstr := fmt.Sprintf("%s:%s@%s/%s?charset=utf8&parseTime=true", user, pwd, host, dbname)
	return gorm.Open("mysql", connstr)
}
