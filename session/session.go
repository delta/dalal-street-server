package session

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	_ "github.com/go-sql-driver/mysql" // mysql package needs to be included like this as required by sqlx
	lru "github.com/hashicorp/golang-lru"
	"github.com/jmoiron/sqlx"

	"github.com/thakkarparth007/dalal-street-server/utils"
)

const sidLen = 32

var logger *logrus.Entry

// Session provides an interface to access a session of a user
type Session interface {
	GetId() string
	Get(string) (string, bool)
	Set(string, string) error
	Delete(string) error
	Destroy() error
}

// cache is a cache of sessions
var cache *lru.Cache

// variables to store the db credentials
var dbUser string
var dbPass string
var dbName string
var dbHost string

// session implements the Session interface
type session struct {
	Id    string
	mutex sync.RWMutex
	m     map[string]string
}

// Load returns a session from database given the session id
func Load(id string) (Session, error) {
	var l = logger.WithFields(logrus.Fields{
		"method": "Load",
		"id":     id,
	})
	if cache.Contains(id) {
		var cachedInterface, _ = cache.Get(id)
		cachedSession, _ := cachedInterface.(*session)
		return cachedSession, nil
	}
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
	cache.Add(id, sess)
	return sess, nil
}

// New returns a new session
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
	cache.Add(sess.Id, sess)
	return sess, nil
}

// GetId returns the Id of the session
func (sess *session) GetId() string {
	return sess.Id
}

// Set sets a key value pair in the map
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

// Get returns the value of the key provided
func (sess *session) Get(str string) (string, bool) {
	sess.mutex.RLock()
	value, ok := sess.m[str] // return value if found or ok=false if not found
	sess.mutex.RUnlock()
	return value, ok
}

// Delete deletes a particular key in a given session
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

// Destroy deletes the entire session from database
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

// Init initializes the session package
func Init(config *utils.Config) {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "session",
	})

	dbUser = config.DbUser
	dbPass = config.DbPassword
	dbName = config.DbName
	dbHost = config.DbHost

	cache, _ = lru.New(config.CacheSize)

	rand.Seed(time.Now().UnixNano())
}
