package session

import (
	"fmt"
)

type Session struct {
	sessionID string
	m         map[string]string
}

func (session Session) init() {

	size := 6 // change the length of the generated random string here
	rb := make([]byte, size)
	_, err := rand.Read(rb)
	if err != nil {
		fmt.Println(err)
	}
	rs := base64.URLEncoding.EncodeToString(rb)
	fmt.Println(rs)
	session.sessionId = rs
	m := make(map[string]string)

}

func (session Session) set(k string, v string) {
	m[k] = v
}

func (session Session) get(str string) string {
	value, ok := cities[str] // return value if found or ok=false if not found
	if ok {
		fmt.Println("value: ", value)
		return value // return value
	} else {
		fmt.Println("key not found")
		return "NULL"
	}
}

func (session Session) delete() {
	for k, v := range m {
		delete(m, k)
	}
	log.Println("Deleted Map")
}

func (session Session) destroy() {
	delSess, err := db.Prepare("DELETE FROM sessions WHERE sessionID=?")
	if err != nil {
		panic(err.Error())
	}
	delSess.Exec(sessionID)
	log.Println("Deleted from Database" + sessionID)
	defer db.Close()
}

// The first time you access the database, save it
func (session Session) save() {

	length = len(m)
	for k, v := range m {
		createSess, err := db.Prepare("INSERT INTO sessions(sessionID, k, v) VALUES(?,?,?)")
		if err != nil {
			panic(err.Error())
		}
		createSess.Exec(sessionID, k, v)
		log.Println("Saved Value Key" + k + "Value:" + v)
	}
}

// Update from the next time
func (session Session) update() {

	length = len(m)
	for k, v := range m {
		createSess, err := db.Prepare("UPDATE sessions SET k=?, v=? WHERE sessionID=?")
		if err != nil {
			panic(err.Error())
		}
		createSess.Exec(k, v, sessionID)
		log.Println("Updated Key" + k + "Value:" + v)
	}
}
