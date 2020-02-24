package session

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	lru "github.com/hashicorp/golang-lru"

	"github.com/delta/dalal-street-server/utils"
)

const sidLen = 32

var logger *logrus.Entry

// Session provides an interface to access a session of a user
type Session interface {
	GetID() string
	Get(string) (string, bool)
	Set(string, string) error
	Touch() error // updates the LastAccessTime of the session
	Delete(string) error
	Destroy() error
}

// cache is a cache of sessions
var cache *lru.Cache

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

	db := utils.GetDB()

	l.Debugf("SessionId: %s", id)

	var rows []struct {
		Key   string
		Value string
	}

	db = db.Raw("SELECT `key`, `value` FROM Sessions WHERE id=?", id).Scan(&rows)
	if db.Error != nil {
		// db won't give error in case the session doesn't exist
		l.Errorf("Error loading session: '%s'", db.Error)
		return nil, db.Error
	}

	// so we should give that error :)
	if len(rows) == 0 {
		return nil, errors.New("Invalid session ID")
	}

	results = make(map[string]string)
	for _, row := range rows {
		results[row.Key] = row.Value
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

	sess.Set("CreatedAt", utils.GetCurrentTimeISO8601())
	sess.Set("LastAccessTime", utils.GetCurrentTimeISO8601())

	l.Debugf("Created session: %+v", sess)
	cache.Add(sess.Id, sess)
	return sess, nil
}

func (sess *session) String() string {
	sess.mutex.RLock()
	defer sess.mutex.RUnlock()

	return fmt.Sprintf("Session[Id=%s, %+v]", sess.Id, sess.m)
}

// GetId returns the ID of the session
func (sess *session) GetID() string {
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

	db := utils.GetDB()

	db = db.Exec("INSERT INTO Sessions VALUES (?,?,?) ON DUPLICATE KEY UPDATE `id`=?, `key`=?, `value`=?",
		sess.Id, k, v, sess.Id, k, v)

	if db.Error != nil {
		l.Errorf("Error in setting-value query: '%s'", db.Error)
		return db.Error
	}

	l.Debugf("Set key in database")
	sess.mutex.Lock()
	defer sess.mutex.Unlock()
	if sess.m == nil {
		l.Errorf("Unable to set session, already destroyed")
		return fmt.Errorf("Map in session does not exist")
	}
	sess.m[k] = v
	return nil
}

// Get returns the value of the key provided
func (sess *session) Get(str string) (string, bool) {
	sess.mutex.RLock()
	value, ok := sess.m[str] // return value if found or ok=false if not found
	sess.mutex.RUnlock()
	return value, ok
}

func (sess *session) Touch() error {
	return sess.Set("LastAccessTime", utils.GetCurrentTimeISO8601())
}

// Delete deletes a particular key in a given session
func (sess *session) Delete(str string) error {
	var l = logger.WithFields(logrus.Fields{
		"method":  "Delete",
		"session": fmt.Sprintf("%v", sess),
		"str":     str,
	})
	l.Debug("Deleting")

	db := utils.GetDB()
	db = db.Exec("Delete FROM Sessions WHERE id=? AND `key`=?", sess.Id, str)

	if db.Error != nil {
		return db.Error
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

	db := utils.GetDB()

	db = db.Exec("DELETE FROM Sessions WHERE id=?", sess.Id)
	if db.Error != nil {
		return db.Error
	}

	sess.mutex.Lock()
	sess.m = nil
	sess.mutex.Unlock()
	cache.Remove(sess.Id)

	return nil
}

// Init initializes the session package
func Init(config *utils.Config) {
	logger = utils.Logger.WithFields(logrus.Fields{
		"module": "session",
	})

	cache, _ = lru.New(config.CacheSize)

	rand.Seed(time.Now().UnixNano())
}
