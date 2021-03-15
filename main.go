package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/delta/dalal-street-server/datastreams"
	"github.com/delta/dalal-street-server/grpcapi"
	"github.com/delta/dalal-street-server/httpapi"
	"github.com/delta/dalal-street-server/matchingengine"
	"github.com/delta/dalal-street-server/models"
	"github.com/delta/dalal-street-server/session"
	"github.com/delta/dalal-street-server/socketapi"
	"github.com/delta/dalal-street-server/utils"
)

func RealMain() {
	config := utils.GetConfiguration()
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Error: '%+v'\n", r)
		}
		utils.CloseDB()
	}()

	if config.Stage != "prod" {
		fmt.Println("WARNING: Server not running in prod stage.")
	}

	utils.Init(config)
	datastreams.Init(config)
	matchingengine.Init(config)
	session.Init(config)
	socketapi.Init(config)

	// handle streams
	datastreamsManager := datastreams.GetManager()
	go datastreamsManager.GetStockExchangeStream().Run()
	go datastreamsManager.GetStockPricesStream().Run()

	models.Init(config, datastreamsManager)
	go models.UpdateLeaderboardTicker()

	matchingEngine := matchingengine.NewMatchingEngine(datastreamsManager)
	grpcapi.Init(config, matchingEngine, datastreamsManager)

	if !utils.IsDockerEnv() {
		models.OpenMarket(false)
	}

	httpServer := http.Server{
		Addr: config.ServerPort,
		Handler: http.HandlerFunc(
			func(resp http.ResponseWriter, req *http.Request) {
				start := time.Now()
				defer func() {
					diff := time.Now().Sub(start).Seconds()
					if r := recover(); r != nil {
						utils.Logger.Errorf("[%.3f] %s %s Error: %+v", diff, req.Method, req.URL.Path, r)
					}
					utils.Logger.Infof("[%.3f] %s %s", diff, req.Method, req.URL.Path)
				}()

				if req.Method == http.MethodOptions && config.Stage != "Prod" {
					resp.Header().Add("Access-Control-Allow-Origin", "*")
					resp.Header().Add("Access-Control-Allow-Methods", "*")
					resp.Header().Add("Access-Control-Allow-Headers", "Content-Type,x-grpc-web,sessionid")
					resp.Header().Add("Access-Control-Max-Age", "600")
					resp.Write([]byte("OK"))
					return
				}
				if utils.IsGrpcRequest(req) {
					grpcapi.GrpcHandlerFunc(resp, req)
				} else if req.URL.Path == "/ws" {
					socketapi.Handle(resp, req)
				} else if req.URL.Path == "/verify" {
					if err := httpapi.HandleVerification(req); err != nil {
						respText := fmt.Sprintf("%s", err.Error())
						resp.Write([]byte(respText))
					} else {
						resp.Write([]byte("Successfully verified account!"))
					}
				} else {
					resp.WriteHeader(http.StatusBadRequest)
					resp.Write([]byte("Invalid URL requested"))
				}
			},
		),
	}
	models.InspectComponents()
	utils.Logger.Fatal(httpServer.ListenAndServeTLS(config.TLSCert, config.TLSKey))
}

func main() {
	for {
		RealMain()
	}
}
