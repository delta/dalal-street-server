package main

import (
	"fmt"
	"net/http"

	"github.com/thakkarparth007/dalal-street-server/utils"
	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/session"
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
