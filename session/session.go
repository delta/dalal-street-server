package session

import (
	"encoding/base64"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"math/rand"
)

type Session struct {
	Id string
	m  map[string]string
}

func GetSession(id string) (Session, error) {
	var session Session
	var results map[string]string
	db, err := dbConn()
	if err != nil {
		return session, err
	}

	rows, err := db.Queryx("SELECT `key`, `value` FROM Sessions WHERE id=?", id)
	if err != nil {
		return session, err
	}

	results = make(map[string]string)

	for rows.Next() {
		var key, value string
		err = rows.Scan(&key, &value)
		if err != nil {
			return session, err
		}
		results[key] = value
	}

	session.Id = id
	session.m = results
	return session, err
}

func NewSession() (Session, error) {
	var session Session
	size := 6 // change the length of the generated random string here
	rb := make([]byte, size)
	_, err := rand.Read(rb)
	if err != nil {
		return session, err
	}
	rs := base64.URLEncoding.EncodeToString(rb)
	session.Id = rs
	session.m = make(map[string]string)
	return session, err
}

// Set a key value pair in the map
func (session Session) Set(k string, v string) error {
	db, err := dbConn()

	if err != nil {
		return err
	}

	sql := "INSERT INTO Sessions VALUES (?,?,?) ON DUPLICATE KEY UPDATE `id`=?, `key`=?, `value`=?"
	_, err = db.Exec(sql, session.Id, k, v, session.Id, k, v)

	if err != nil {
		return err
	}

	session.m[k] = v
	return nil
}

// Get the value providing key to the get function
func (session Session) Get(str string) (string, bool) {
	value, ok := session.m[str] // return value if found or ok=false if not found
	return value, ok
}

func (session Session) Delete(str string) error {
	db, err := dbConn()

	if err != nil {
		return err
	}

	sql := "Delete FROM Sessions WHERE id=? AND `key`=?"
	_, err = db.Exec(sql, session.Id, str)

	return err
}

// Delete the entire session from database
func (session Session) Destroy() error {
	db, err := dbConn()

	if err != nil {
		return err
	}

	sql := "DELETE FROM Sessions WHERE id=?"
	_, err = db.Exec(sql, session.Id)

	return err
}

func dbConn() (*sqlx.DB, error) {
	var db *sqlx.DB
	dbDriver := "mysql" // Database driver
	dbUser := "root"    // Mysql username
	dbPass := ""        // Mysql password
	dbName := "gotest"  // Mysql schema
	db, err := sqlx.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	if err != nil {
		return db, err
	}
	return db, err
}
