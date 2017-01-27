package session

import (
	"fmt"
	"math/rand"

	"github.com/jmoiron/sqlx"
	_ "github.com/go-sql-driver/mysql"
	"github.com/Sirupsen/logrus"

	"github.com/thakkarparth007/dalal-street-server/utils"
)

const sidLen = 32

var (
	dbUser string
	dbPass string
	dbHost string
	dbName string

	logger *logrus.Entry
)

type Session struct {
	Id string
	m  map[string]string
}

func Load(id string) (*Session, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Load",
		"id": id,
	})

	var (
		session *Session
		results map[string]string
	)

	db, err := dbConn()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	l.Debugf("SessionId: %s", id)
	rows, err := db.Queryx("SELECT `key`, `value` FROM Sessions WHERE id=?", id)
	if err != nil {
		l.Errorf("Error loading session: '%s'", err)
		return nil, err
	}

	results = make(map[string]string)

	for rows.Next() {
		var key, value string
		err = rows.Scan(&key, &value)
		if err != nil {
			return nil, err
		}
		results[key] = value
	}

	session.Id = id
	session.m = results

	l.Debugf("Loaded session: %+v", session)
	return session, nil
}

func New() (*Session, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "New",
	})

	var session *Session

	rb := make([]byte, sidLen)
	_, err := rand.Read(rb)
	if err != nil {
		return nil, err
	}

	session.Id = string(rb)
	session.m = make(map[string]string)

	l.Debugf("Created session: %+v", session)
	return session, nil
}

// Set a key value pair in the map
func (session *Session) Set(k string, v string) error {
	var l = logger.WithFields(logrus.Fields{
		"method": "Set",
		"session": fmt.Sprintf("%v", session),
		"k": k,
		"v": v,
	})

	db, err := dbConn()
	if err != nil {
		return err
	}
	defer db.Close()

	sql := "INSERT INTO Sessions VALUES (?,?,?) ON DUPLICATE KEY UPDATE `id`=?, `key`=?, `value`=?"
	_, err = db.Exec(sql, session.Id, k, v, session.Id, k, v)

	if err != nil {
		l.Errorf("Error in setting-value query: '%s'", err)
		return err
	}

	l.Debugf("Set key in database")
	session.m[k] = v
	return nil
}

// Get the value providing key to the get function
func (session *Session) Get(str string) (string, bool) {
	value, ok := session.m[str] // return value if found or ok=false if not found
	return value, ok
}

func (session *Session) Delete(str string) error {
	var l = logger.WithFields(logrus.Fields{
		"method": "Delete",
		"session": fmt.Sprintf("%v", session),
		"str": str,
	})
	l.Debug("Deleting")

	db, err := dbConn()
	if err != nil {
		return err
	}
	defer db.Close()

	sql := "Delete FROM Sessions WHERE id=? AND `key`=?"
	_, err = db.Exec(sql, session.Id, str)

	return err
}

// Delete the entire session from database
func (session Session) Destroy() error {
	var l = logger.WithFields(logrus.Fields{
		"method": "Delete",
		"session": fmt.Sprintf("%v", session),
	})
	l.Debug("Destroying")

	db, err := dbConn()
	if err != nil {
		return err
	}
	defer db.Close()

	sql := "DELETE FROM Sessions WHERE id=?"
	_, err = db.Exec(sql, session.Id)

	return err
}

func dbConn() (*sqlx.DB, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "dbConn",
	})

	var db *sqlx.DB
	db, err := sqlx.Open("mysql", dbUser+":"+dbPass+"@"+dbHost+"/"+dbName)
	if err != nil {
		l.Errorf("Error opening database: '%s'", err)
	}
	return db, err
}

func InitSession() {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "session",
	})
	dbUser = utils.Configuration.DbUser
	dbPass = utils.Configuration.DbPassword
	dbName = utils.Configuration.DbName
	dbHost = utils.Configuration.DbHost
}
