package main

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
    "log"
)

func Create(id int, email  string , name string) {
    db := dbConn()

    createSess, err := db.Prepare("INSERT INTO sessions(name, email) VALUES(?,?)")

    if err != nil {
        panic(err.Error())
    }

    createSess.Exec(name, email)
    log.Println("INSERT: Name: " + name + " | E-mail: " + email)
    defer db.Close()
}

func ReadAll() []Sessions {
    db := dbConn()
    selDB, err := db.Query("SELECT * FROM sessions ORDER BY id DESC")
    if err != nil {
        panic(err.Error())
    }
    n := Sessions{}
    narr := []Sessions{}
    for selDB.Next() {
        var id int
        var name, email string
        err = selDB.Scan(&id, &name, &email)
        if err != nil {
            panic(err.Error())
        }
        n.Id = id
        n.Name = name
        n.Email = email
        narr = append(narr, n)
    }
    defer db.Close()
    return narr
}

func Read(id int) Sessions {

    db := dbConn()
    selDB, err := db.Query("SELECT * FROM sessions WHERE id=?", id)
    if err != nil {
        panic(err.Error())
    }
    n := Sessions{}
    for selDB.Next() {
        var id int
        var name, email string
        err = selDB.Scan(&id, &name, &email)
        if err != nil {
            panic(err.Error())
        }
        n.Id = id
        n.Name = name
        n.Email = email
    }
    log.Println(n.Name)
    defer db.Close()
    return n
}

func Update(id int, email  string , name string) {

    db := dbConn()

    createSess, err := db.Prepare("UPDATE sessions SET name=?, email=? WHERE id=?")

    if err != nil {
        panic(err.Error())
    }
    createSess.Exec(name, email, id)
    log.Println("UPDATE: Name: " + name + " | E-mail: " + email)
    defer db.Close()
}

func Destroy(id int) {

    db := dbConn()
    delSess, err := db.Prepare("DELETE FROM sessions WHERE id=?")
    if err != nil {
        panic(err.Error())
    }
    delSess.Exec(id)
    log.Println("DELETE")
    defer db.Close()
}

type Sessions struct {
    Id    int
    Name  string
    Email string
}

func main() {
//    Create(1, "john@ymail.com", "John")
//    Create(2, "random@ymail.com", "Random")

/*    var res Sessions
      res = Read(2)
      log.Println(res.Id)

      var res2 []Sessions
      res2 = ReadAll()
      log.Println(res2) */

//    Destroy(2)

}

func dbConn() (db *sql.DB) {
	dbDriver := "mysql"   // Database driver
	dbUser := "root"      // Mysql username
	dbPass := "" // Mysql password
	dbName := "gotest"   // Mysql schema

	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)

	if err != nil {
		panic(err.Error())
	}
	return db
}
