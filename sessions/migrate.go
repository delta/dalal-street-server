package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	db, err := sql.Open("mysql", "root@(127.0.0.1:3306)/gotest")

	if err != nil {
		fmt.Print(err.Error())
	}

	defer db.Close()

	err = db.Ping()
	if err != nil {
		fmt.Print(err.Error())
	}
	// Create table session with the required parameters
	stmt, err := db.Prepare("CREATE TABLE sessions(id int(10) NOT NULL AUTO_INCREMENT, name varchar(40) NOT NULL, email varchar(80) NOT NULL, PRIMARY KEY (id));")

	if err != nil {
		fmt.Println(err.Error())
	}
	_, err = stmt.Exec()

	if err != nil {
		fmt.Print(err.Error())
	} else {
		fmt.Printf("Sessions Table successfully migrated....")
	}
}
