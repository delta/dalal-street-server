package main

import (
	"fmt"
	"net/http"

	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/socketapi"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

func RealMain() {
	//utils.InitConfiguration("config.json")
	defer func() {
		recover()
	}()

	if utils.Configuration.Stage != "prod" {
		fmt.Println("WARNING: Server not running in prod stage.")
	}

	//utils.InitLogger()

	models.InitMatchingEngine()
	//models.InitModels()
	//session.InitSession()
	//socketapi.InitSocketApi()

	go models.UpdateLeaderboardTicker()

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/ws", socketapi.Handle)

	port := fmt.Sprintf(":%d", utils.Configuration.HttpPort)
	utils.Logger.Fatal(http.ListenAndServe(port, nil))
}

func main() {
	for {
		RealMain()
	}
}
