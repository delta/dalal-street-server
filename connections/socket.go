package connections

import (
	"github.com/gorilla/websocket"
	"net/http"
	"log"
)
var Test = make(chan interface{},3)
var upgrader = websocket.Upgrader{}
var mesageChan = make(chan interface{}, 10)

func WebsocketServerInit() {
	http.HandleFunc("/listener/ws", func(w http.ResponseWriter,r *http.Request) {
		conn,_ := upgrader.Upgrade(w,r,nil)
		go listener(conn)
	})
	http.HandleFunc("/writer/ws", func(w http.ResponseWriter, r *http.Request) {
		conn,_ := upgrader.Upgrade(w,r,nil)
		go writer(conn)
	})
	http.ListenAndServe(":3000",nil)
}

func SendMessage(v interface{} , id int)  {
	switch id {
	default:
		log.Print("sending msg")
		mesageChan <- v
	}
}
func listener(conn *websocket.Conn)  {
	for{
		mType,msg,_ := conn.ReadMessage()
		log.Println("message Type :",mType," message :",string(msg))
		Test <- 1
		log.Print(Test)
		if mType == -1 {
			log.Println("connection closed")
			break
		}
	}
}

func writer(conn *websocket.Conn) {
	for {
		log.Print("waiting to write")
		conn.WriteJSON(<-mesageChan)
		log.Print("fineshed writing")
	}
}

