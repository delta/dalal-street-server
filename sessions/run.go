package session

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

type Session struct {
	parameter1 int
	parameter2 string
	parameter3 string
	parameter4 string
}

func getSession(parameter1 int) Session {

	db := dbConn()
	selDB, err := db.Query("SELECT * FROM sessions WHERE parameter1=?", parameter1)
	if err != nil {
		panic(err.Error())
	}
	n := Session{}
	for selDB.Next() {
		var parameter1 int
		var parameter2, parameter3 string
		err = selDB.Scan(&parameter1, &parameter2, &parameter3)
		if err != nil {
			panic(err.Error())
		}
		n.parameter1 = parameter1 //put parameters in caps here to return json properly
		n.parameter2 = parameter2
		n.parameter3 = parameter3
	}
	log.Println(n.parameter2)
	defer db.Close()
	return n

}

func getparam2(stringsearch string) Session {

	db := dbConn()
	selDB, err := db.Query("SELECT * FROM sessions WHERE parameter2=?", stringsearch)
	if err != nil {
		panic(err.Error())
	}
	n := Session{}
	for selDB.Next() {
		var parameter1 int
		var parameter2, parameter3 string
		err = selDB.Scan(&parameter1, &parameter2, &parameter3)
		if err != nil {
			panic(err.Error())
		}
		n.parameter1 = parameter1 //put parameters in caps here to return json properly
		n.parameter2 = parameter2
		n.parameter3 = parameter3
	}
	return n
}

func getparam3(stringsearch string) Session {

	db := dbConn()
	selDB, err := db.Query("SELECT * FROM sessions WHERE parameter3=?", stringsearch)
	if err != nil {
		panic(err.Error())
	}
	n := Session{}
	for selDB.Next() {
		var parameter1 int
		var parameter2, parameter3 string
		err = selDB.Scan(&parameter1, &parameter2, &parameter3)
		if err != nil {
			panic(err.Error())
		}
		n.parameter1 = parameter1 //put parameters in caps here to return json properly
		n.parameter2 = parameter2
		n.parameter3 = parameter3
	}
	return n
}

func getparam4(stringsearch string) Session {

	db := dbConn()
	selDB, err := db.Query("SELECT * FROM sessions WHERE parameter4=?", stringsearch)
	if err != nil {
		panic(err.Error())
	}
	n := Session{}
	for selDB.Next() {
		var parameter1 int
		var parameter2, parameter3 string
		err = selDB.Scan(&parameter1, &parameter2, &parameter3)
		if err != nil {
			panic(err.Error())
		}
		n.parameter1 = parameter1 //put parameters in caps here to return json properly
		n.parameter2 = parameter2
		n.parameter3 = parameter3
	}
	return n
}

func Setparam2(session Session, strinput string) {
	session.parameter2 = strinput
}
func Setparam3(session Session, strinput string) {
	session.parameter3 = strinput
}
func Setparam4(session Session, strinput string) {
	session.parameter4 = strinput
}

func Save(session Session) {

	parameter1 = session.parameter1
	parameter2 = session.parameter2
	parameter3 = session.parameter3
	parameter4 = session.parameter4
	db := dbConn()
	createSess, err := db.Prepare("UPDATE sessions SET parameter2=?, parameter3=?, parameter4=? WHERE parameter1=?")
	if err != nil {
		panic(err.Error())
	}
	createSess.Exec(parameter4, parameter3, parameter2, parameter1)
	log.Println("SAVE: parameter2: " + parameter2 + " parameter3: " + parameter3 + " parameter4: " + parameter4)
	defer db.Close()

}

func Destroy(session Session) {

	parameter1 = session.parameter1
	db := dbConn()
	delSess, err := db.Prepare("DELETE FROM sessions WHERE parameter1=?")
	if err != nil {
		panic(err.Error())
	}
	delSess.Exec(parameter1)
	log.Println("DELETE")
	defer db.Close()

}

func dbConn() (db *sql.DB) {

	dbDriver := "mysql" // Database driver
	dbUser := "root"    // Mysql username
	dbPass := ""        // Mysql password
	dbName := "gotest"  // Mysql schema
	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		panic(err.Error())
	}
	return db

}
