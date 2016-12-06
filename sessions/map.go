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
func (session Session) destroy() {
	for k, v := range m {
		delete(m, k)
	}
}
