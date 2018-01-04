package session

import (
	"encoding/base64"
	"math/rand"
	"sync"
)

type fakeSession struct {
	Id    string
	mutex sync.RWMutex
	m     map[string]string
}

func Fake() (Session, error) {
	var sess = &fakeSession{}

	rb := make([]byte, sidLen)
	_, err := rand.Read(rb)
	if err != nil {
		return nil, err
	}

	sess.Id = base64.URLEncoding.EncodeToString(rb)
	sess.m = make(map[string]string)

	return sess, nil
}

func (sess *fakeSession) GetId() string {
	return sess.Id
}

func (sess *fakeSession) Get(str string) (string, bool) {
	sess.mutex.RLock()
	value, ok := sess.m[str] // return value if found or ok=false if not found
	sess.mutex.RUnlock()
	return value, ok
}

func (sess *fakeSession) Set(k string, v string) error {
	sess.mutex.Lock()
	sess.m[k] = v
	sess.mutex.Unlock()
	return nil
}

func (sess *fakeSession) Delete(str string) error {
	return nil
}

func (sess *fakeSession) Destroy() error {
	return nil
}
