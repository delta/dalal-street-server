package session

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

func Create(parameter1 int, parameter3 string, parameter2 string) {
	db := dbConn()
	createSess, err := db.Prepare("INSERT INTO sessions(parameter2, parameter3) VALUES(?,?)")
	if err != nil {
		panic(err.Error())
	}
	createSess.Exec(parameter2, parameter3)
	log.Println("INSERT: parameter2: " + parameter2 + " | E-mail: " + parameter3)
	defer db.Close()

}

func Read(parameter1 int) Session {

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

// Save by searching for the element (key-value pair) using parameter1 and update the other 3
func Save(session Session, locsession LocSession) Session {

	parameter1 = locsession.parameter1
	parameter2 = locsession.parameter2
	parameter3 = locsession.parameter3
	parameter4 = locsession.parameter4
	db := dbConn()
	createSess, err := db.Prepare("UPDATE sessions SET parameter2=?, parameter3=?, parameter4=? WHERE parameter1=?")
	if err != nil {
		panic(err.Error())
	}
	createSess.Exec(parameter2, parameter3, parameter1)
	log.Println("SAVE: parameter2: " + parameter2 + " parameter3: " + parameter3 + " parameter4: " + parameter4)
	defer db.Close()
	session.parameter1 = locsession.parameter1
	session.parameter2 = locsession.parameter2
	session.parameter3 = locsession.parameter3
	session.parameter4 = locsession.parameter4
	return session

}

func Updateparam2(session LocSession, parameter2 string) {
	session.parameter2 = parameter2
}
func Updateparam3(session LocSession, parameter3 string) {
	session.parameter3 = parameter3
}
func Updateparam2(session LocSession, parameter4 string) {
	session.parameter4 = parameter4
}

func Destroy(parameter1 int) {

	db := dbConn()
	delSess, err := db.Prepare("DELETE FROM sessions WHERE parameter1=?")
	if err != nil {
		panic(err.Error())
	}
	delSess.Exec(parameter1)
	log.Println("DELETE")
	defer db.Close()

}

//here paramter 1 is used as the primary key
// Add more parameters if required
// Rename parameters to their respective names
type Session struct {
	parameter1 int
	parameter2 string
	parameter3 string
	parameter4 string
}

type LocSession struct {
	parameter1 int
	parameter2 string
	parameter3 string
	parameter4 string
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
