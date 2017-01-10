package main
// This main is for testing sockets only so ignore it
import(
	"./connections"
	"log"
)

func main() {
	go func() {
		log.Print("starting go loop")
		s := <- connections.Test
		log.Print("ending go loop")
		connections.SendMessage(zold{ Name: "rahul"}, 1)
		log.Print(s)
	}()
	connections.WebsocketServerInit()
}
type zold struct {
	Name string
}
