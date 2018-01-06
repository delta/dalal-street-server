package main

import (
	"fmt"
	"net/http"

	"github.com/thakkarparth007/dalal-street-server/datastreams"
	"github.com/thakkarparth007/dalal-street-server/grpcapi"
	"github.com/thakkarparth007/dalal-street-server/matchingengine"
	"github.com/thakkarparth007/dalal-street-server/models"
	"github.com/thakkarparth007/dalal-street-server/session"
	"github.com/thakkarparth007/dalal-street-server/socketapi"
	"github.com/thakkarparth007/dalal-street-server/utils"
)

func RealMain() {
	config := utils.GetConfiguration()

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Error: '%+v'\n", r)
		}
	}()

	if config.Stage != "prod" {
		fmt.Println("WARNING: Server not running in prod stage.")
	}

	utils.Init(config)
	datastreams.Init(config)
	grpcapi.Init(config)
	matchingengine.Init(config)
	session.Init(config)
	socketapi.Init(config)

	// handle streams
	datastreamsManager := datastreams.GetManager()
	go datastreamsManager.GetStockExchangeStream().Run()
	go datastreamsManager.GetStockPricesStream().Run()

	models.Init(config, datastreamsManager)

	matchingEngine := matchingengine.NewMatchingEngine(datastreamsManager)
	grpcapi.StartServices(matchingEngine, datastreamsManager)

	httpServer := http.Server{
		Addr: config.GrpcAddress,
		Handler: http.HandlerFunc(grpcapi.GrpcHandlerFunc),
	}

	go func() {
		err := httpServer.ListenAndServeTLS(config.GrpcCert, config.GrpcKey)
		if err != nil {
			utils.Logger.Fatalf("Failed while starting server. Error: %+v", err)
		}
	}()
	//models.InitModels()
	//session.InitSession()
	//socketapi.InitSocketApi()

	go models.UpdateLeaderboardTicker()

	http.Handle("/", http.FileServer(http.Dir("./public")))
	http.HandleFunc("/ws", socketapi.Handle)

	port := fmt.Sprintf(":%d", config.HttpPort)
	utils.Logger.Fatal(http.ListenAndServe(port, nil))

	for {

	}
}

func main() {
	for {
		RealMain()
	}
}
