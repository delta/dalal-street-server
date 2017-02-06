package main

import (
	"fmt"
	"net/http"

	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/session"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

func main() {
	utils.InitConfiguration("config.json")
	utils.InitLogger()

	models.InitModels()

	session.InitSession()

	http.Handle("/", http.FileServer(http.Dir("./public")))

	port := fmt.Sprintf(":%d", utils.Configuration.HttpPort)
	utils.Logger.Fatal(http.ListenAndServe(port, nil))
}
