package session

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

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

type Session interface {
	GetId() string
	Get(string) (string, bool)
	Set(string, string) error
	Delete(string) error
	Destroy() error
}

type session struct {
	Id    string
	mutex sync.RWMutex
	m     map[string]string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func Load(id string) (Session, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Load",
		"id":     id,
	})

	var (
		sess    = &session{}
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

	sess.Id = id
	sess.m = results

	l.Debugf("Loaded session: %+v", sess)
	return sess, nil
}

func New() (Session, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "New",
	})

	var sess = &session{}

	rb := make([]byte, sidLen)
	_, err := rand.Read(rb)
	if err != nil {
		return nil, err
	}

	sess.Id = base64.URLEncoding.EncodeToString(rb)
	sess.m = make(map[string]string)

	l.Debugf("Created session: %+v", sess)
	return sess, nil
}

func (sess *session) GetId() string {
	return sess.Id
}

// Set a key value pair in the map
func (sess *session) Set(k string, v string) error {
	var l = logger.WithFields(logrus.Fields{
		"method":  "Set",
		"session": fmt.Sprintf("%v", sess),
		"k":       k,
		"v":       v,
	})

	db, err := dbConn()
	if err != nil {
		return err
	}
	defer db.Close()

	sql := "INSERT INTO Sessions VALUES (?,?,?) ON DUPLICATE KEY UPDATE `id`=?, `key`=?, `value`=?"
	_, err = db.Exec(sql, sess.Id, k, v, sess.Id, k, v)

	if err != nil {
		l.Errorf("Error in setting-value query: '%s'", err)
		return err
	}

	l.Debugf("Set key in database")
	sess.mutex.Lock()
	sess.m[k] = v
	sess.mutex.Unlock()
	return nil
}

// Get the value providing key to the get function
func (sess *session) Get(str string) (string, bool) {
	sess.mutex.RLock()
	value, ok := sess.m[str] // return value if found or ok=false if not found
	sess.mutex.RUnlock()
	return value, ok
}

func (sess *session) Delete(str string) error {
	var l = logger.WithFields(logrus.Fields{
		"method":  "Delete",
		"session": fmt.Sprintf("%v", sess),
		"str":     str,
	})
	l.Debug("Deleting")

	db, err := dbConn()
	if err != nil {
		return err
	}
	defer db.Close()

	sql := "Delete FROM Sessions WHERE id=? AND `key`=?"
	_, err = db.Exec(sql, sess.Id, str)

	if err != nil {
		return err
	}

	sess.mutex.Lock()
	delete(sess.m, str)
	sess.mutex.Unlock()

	return nil
}

// Delete the entire session from database
func (sess *session) Destroy() error {
	var l = logger.WithFields(logrus.Fields{
		"method":  "Delete",
		"session": fmt.Sprintf("%v", sess),
	})
	l.Debug("Destroying")

	db, err := dbConn()
	if err != nil {
		return err
	}
	defer db.Close()

	sql := "DELETE FROM Sessions WHERE id=?"
	_, err = db.Exec(sql, sess.Id)
	if err != nil {
		return err
	}

	sess.m = make(map[string]string)

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

func init() {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "session",
	})
	dbUser = utils.Configuration.DbUser
	dbPass = utils.Configuration.DbPassword
	dbName = utils.Configuration.DbName
	dbHost = utils.Configuration.DbHost
}
