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
	go models.UpdateLeaderboardTicker()

	models.Init(config, datastreamsManager)

	matchingEngine := matchingengine.NewMatchingEngine(datastreamsManager)
	grpcapi.StartServices(matchingEngine, datastreamsManager)

	httpServer := http.Server{
		Addr: config.ServerPort,
		Handler: http.HandlerFunc(
			func (resp http.ResponseWriter, req *http.Request) {
				if req.Method == http.MethodOptions && config.Stage != "Prod" {
					resp.Header().Add("Access-Control-Allow-Origin", "*")
					resp.Header().Add("Access-Control-Allow-Methods", "*")
					resp.Header().Add("Access-Control-Allow-Headers", "Content-Type,x-grpc-web")
					resp.Write([]byte("OK"))
					return
				}
				if utils.IsGrpcRequest(req) {
					grpcapi.GrpcHandlerFunc(resp, req)
				} else {
					socketapi.Handle(resp, req)
				}
			},
		),
	}

	utils.Logger.Fatal(httpServer.ListenAndServeTLS(config.TLSCert, config.TLSKey))
	
	//models.InitModels()
	//session.InitSession()
	//socketapi.InitSocketApi()


	// http.Handle("/", http.FileServer(http.Dir("./public")))
	// http.HandleFunc("/ws", socketapi.Handle)

	// utils.Logger.Fatal(http.ListenAndServe(config.ServerPort, nil))

	for {

	}
}

func main() {
	for {
		RealMain()
	}
}
